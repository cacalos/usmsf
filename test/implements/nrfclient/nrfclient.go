package nrfclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"os"
	"sync"
	"time"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/utils"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"

	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/uregi"

	"github.com/go-openapi/strfmt"
)

// Internal flow control signals
const (
	_sigTerminateNRFClient = iota
	_sigDoneNRFClient
)

// NRFClientServiceName Trace 출력 용 서비스 이름
const NRFClientServiceName = "NRFClient"

var loggers = common.SamsungLoggers()

// TODO hanmouse: delete
//var logger = ulog.GetLogger("com.uangel.usmsf.nrfclient")

// NRFClient NRF 연동 클라이언트
type NRFClient struct {
	httpClient     uclient.HTTP
	http2c         uclient.H2C
	circuitBreaker uclient.HTTPCircuitBreaker
	resolver       uregi.Resolver
	router         uclient.HTTPRouter

	nfInstancesURI            string
	subscriptionsURI          string
	nfStatusNotifURI          string
	nfProfile                 *msg5g.NFMgmtProfile
	running                   bool
	NfRegistered              bool
	retryRegisterInterval     int
	retryRegisterCount        int
	heartBeatInterval         int
	heartBeatFailureThreshold int
	termSignal                chan int
	doneSignal                chan int
	waitGroup                 sync.WaitGroup
	pcfInfoTable              *PcfInfoTable
	validityTimeExtention     time.Duration
	terminating               bool
	traceMgr                  interfaces.TraceMgr
}

func (c *NRFClient) InitConfig(cfg uconf.Config) {
	retryRegisterInterval := cfg.GetInt("nrf-client.nrf-registration.retry-register-interval", 30)
	heartBeatInterval := cfg.GetInt("nrf-client.nrf-registration.heart-beat-interval", 60)
	heartBeatFailureThreshold := cfg.GetInt("nrf-client.nrf-registration.heart-beat-failure-threshold", 5)
	validityTimeExtention := time.Duration(cfg.GetInt("nrf-client.nrf-registration.validity-time-extention", 480)) * time.Minute

	c.retryRegisterInterval = retryRegisterInterval
	c.heartBeatInterval = heartBeatInterval
	c.heartBeatFailureThreshold = heartBeatFailureThreshold
	c.validityTimeExtention = validityTimeExtention
	c.retryRegisterCount = 0

}

func (c *NRFClient) InitRegiConfig(cfg uconf.Config) (err error) {

	// Get API from .conf

	conf_reg := "nrf-client.nrf-registration.target-nfi-uri"
	if uri, err := getURIConfig(cfg, conf_reg); err == nil { //nf-instance
		c.nfInstancesURI = uri
	} else {
		loggers.ErrorLogger().Major("A required item was not configured: %s", conf_reg)
		return err
	}

	return err

}

func (c *NRFClient) InitReRegiConfig(cfg uconf.Config) (err error) {

	// Get full URI for /nnrf-nfm/v1/subscriptions API
	configPath := "nrf-client.nrf-registration.target-subs-uri"
	if uri, err := getURIConfig(cfg, configPath); err == nil {
		c.subscriptionsURI = uri
	} else {
		loggers.ErrorLogger().Major("A required item was not configured: %s", configPath)
		return nil
	}
	// tls 설정이 존재하면, HTTP/2로 통신 하도록 설정
	/*
		configPath = "nrf-client.nrf-registration.tls"
		tlsConfig := cfg.GetConfig(configPath)
		if tlsConfig != nil {
			ci, err := common.NewCertInfoByCfg(tlsConfig)
			if err != nil {
				logger.With(ulog.Fields{"error": err.Error()}).Error("Failed to create CertInfo instance")
				return nil
			}
			c.httpClient.Transport = &http2.Transport{
				TLSClientConfig: ci.GetClientTLSConfig(),
			}
		} else { // HTTP/1.1 사용해야 하는 경우
			c.httpClient.Transport = &http.Transport{}
		}
	*/
	return err
}

func (c *NRFClient) GetMyProfile(cfg uconf.Config) error {
	configPath := "nrf-client.nrf-registration.profile"
	path := cfg.GetString(configPath, "none") //json 읽어서 값 넣는거다..

	if path == "none" {
		loggers.ErrorLogger().Major("Get Fail -> MyInstaceId")
		return errors.New("Get Fail -> MyInstaceId")
	}

	nfProfile, err := msg5g.LoadProfileFrom(path)
	if err != nil {
		loggers.ErrorLogger().Major("Failed to load SMSF profile: error=%#v", err.Error())
		return err
	}
	c.nfProfile = nfProfile

	return err

}

func (c *NRFClient) NRFhostConfig(cfg uconf.Config) (err error) {
	var scheme string
	var client uclient.HTTP

	conf := cfg.GetConfig("nrf-client")
	if conf != nil {
		scheme = conf.GetString("scheme", "h2c")
	} else {
		return errors.New("Need to Config HTTP scheme")
	}

	if scheme == "http" || scheme == "https" {
		client = c.httpClient
		loggers.InfoLogger().Comment("Current scheme : %s", scheme)
	} else if scheme == "h2c" {
		client = c.http2c
		loggers.InfoLogger().Comment("Current scheme : %s", scheme)
	} else {
		return errors.New("Invalid HTTP schme")
	}

	nrfhost := os.Getenv("NRF_HOST")

	c.router = uclient.HTTPRouter{
		Scheme:           scheme,
		Servers:          []string{nrfhost},
		Client:           client,
		CircuitBreaker:   c.circuitBreaker,
		Random:           false,
		RetryStatusCodes: uclient.StatusCodeSet(500, 501, 502, 504, 508),
	}

	return nil

}

// NewNRFClient 새로운 NRFClient 생성 및 반환
func NewNRFClient(
	cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
	httpcli uclient.HTTP, // http or https Interface
	http2c uclient.H2C, // H2C Interface
	circuitBreaker uclient.HTTPCircuitBreaker, // circuitBreaker 구현 Interface 정의
	resolver uregi.Resolver, //service or pod의 IP를 얻어오기 위한 Interface 정의
) *NRFClient {

	//	c := &NRFClient{}

	c := &NRFClient{
		httpClient:     httpcli,
		http2c:         http2c,
		circuitBreaker: circuitBreaker,
		resolver:       resolver,
		running:        false,
		NfRegistered:   false,
		pcfInfoTable:   NewPcfInfoTable(),
		termSignal:     make(chan int),
		doneSignal:     make(chan int),
		waitGroup:      sync.WaitGroup{},
		terminating:    false,
		traceMgr:       traceMgr,
	}

	c.InitConfig(cfg)

	err := c.InitRegiConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	err = c.GetMyProfile(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	err = c.NRFhostConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	err = c.InitReRegiConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	return c
}

// GetBSFProfile BSF의 NFProfile 반환 (NRFClientService 인터페이스 구현)
func (c *NRFClient) GetBSFProfile() *msg5g.NFMgmtProfile {
	return c.nfProfile
}

// SetNFStatusNotifyCallbackURI NFStatus Notify 를 수신할 Callback URI 설정 (NRFClientService 인터페이스 구현)
func (c *NRFClient) SetNFStatusNotifyCallbackURI(uri string) {
	c.nfStatusNotifURI = uri
}

// Start NRF 클라이언트 기동 (NRFClientService 인터페이스 구현)
func (c *NRFClient) Start() {
	if !c.running {
		c.running = true
		exec.SafeGo(c.run)
		//go c.run()
		loggers.InfoLogger().Comment("NRF client has been started.")
	}
}

// Stop NRF 클라이언트 종료 (NRFClientService 인터페이스 구현)
func (c *NRFClient) Stop() {
	if !c.terminating {
		c.terminating = true

		if c.NfRegistered {
			c.termSignal <- _sigTerminateNRFClient
			<-c.doneSignal // waiting for termination of NFRequest or NFHeart-beat

			c.clearAllMonitoringInfo()
			c.waitGroup.Wait() // waiting for terminations of all validity time checking threads

			loggers.ErrorLogger().Minor("NRF Client is sending NFDeregister ...")
			if err := c.sendNFDeregister(); err == nil {
				c.NfRegistered = false
				loggers.InfoLogger().Comment("SMSF has been deregistered from NRF.")
			} else {
				loggers.ErrorLogger().Major("Failed to deregister SMSF: %s", err.Error())
			}

		}
		loggers.InfoLogger().Comment("NRF client has been stopped.")
	} else {
		loggers.ErrorLogger().Minor("NRF client is aready in terminating process.")
	}
}

// Close Eager Instance 종료 시 호출 됨
//func (c *NRFClient) Close() error {
func (c *NRFClient) CloseGracefully() error {
	c.Stop()
	return nil
}

// AddMonitoring PCF에 대한 상태 감시 수행 (NRFClientService 인터페이스 구현)
func (c *NRFClient) AddMonitoring(pcfInstanceID string) {
	pcfInfoTable := c.pcfInfoTable
	if _, ok := pcfInfoTable.Get(pcfInstanceID); !ok {
		pcfInfo := &PcfInfo{
			InstanceID: pcfInstanceID,
			Removing:   make(chan bool),
			// ValidityTime 은 Subscribe 성공 시 하면, NRF에 의해 할당 받음
		}

		if err := c.sendNFStatusSubscribe(pcfInfo); err != nil {
			loggers.ErrorLogger().Major("Failed to subscribe PCF (ID: %s): %s", pcfInstanceID, err.Error())
			return
		}
		pcfInfo.ValidityTimeChecker = c.checkNfStatusNotifyValidityTime(pcfInfo)
		pcfInfoTable.Add(pcfInfo)
		loggers.InfoLogger().Comment("A PCF (ID: %s) has been added and subscribed successfully.", pcfInstanceID)
	} else {
		loggers.ErrorLogger().Minor("A PCF (ID: %s) is already managed for monitoring.", pcfInstanceID)
	}
}

// RemoveMonitoring Deregister 된 PCF에 대한 감시 해제 수행 (NRFClientService 인터페이스 구현)
func (c *NRFClient) RemoveMonitoring(pcfInstanceID string) {
	pcfInfoTable := c.pcfInfoTable

	if pcfInfo, ok := pcfInfoTable.Get(pcfInstanceID); ok {
		if err := c.sendNFStatusUnsubscribe(pcfInfo.SubscriptionID); err != nil {
			loggers.ErrorLogger().Major("Failed to unsubscribe the deregistered PCF (ID: %s): %s", pcfInstanceID, err.Error())
		}

		pcfInfoTable.Remove(pcfInstanceID)
		loggers.InfoLogger().Comment("A deregistered PCF (ID: %s) has been removed from the managed table successfully.", pcfInstanceID)
	} else {
		// 현재 관리되지 않는 ID 임에도 호출 되었다는 건, 이전에 Unsub에 실패
		// 한 것으로 볼 수 있으므로, 다시 한 번 Unsub 수행
		loggers.InfoLogger().Comment("Sending NFStatusUnsubscribe for the unmanaged PCF (ID: %s)", pcfInstanceID)
		if err := c.sendNFStatusUnsubscribe(pcfInstanceID); err != nil {
			loggers.ErrorLogger().Major("Failed to unsubscribe PCF (ID: %s): %s", pcfInstanceID, err.Error())
			return
		}
		loggers.InfoLogger().Comment("The unmanaged PCF (ID: %s) has been unsubscribed succressfully.", pcfInstanceID)
	}
}

// GetRetryRegisterCount NFRegister 시도 횟수를 가져 옴
func (c *NRFClient) GetRetryRegisterCount() int {
	return c.retryRegisterCount
}

// Registered NRF 에 등록 되었는지 확인
func (c *NRFClient) Registered() bool {
	return c.NfRegistered
}

// NRF 클라이언트의 기본 작업(NFRegister, NFHeart-Beat) 수행
func (c *NRFClient) run() {
StartNFRegister:
	if registered := c.registerSMSF(); registered {
		c.NfRegistered = registered
		if err := c.sendNFHeartBeats(); err != nil {
			// too many failures in sending NFHeart-beat
			loggers.ErrorLogger().Minor("No more NFHeart-beat: %s", err.Error())
			loggers.ErrorLogger().Minor("Changing state to NFRegistering ...")
			c.NfRegistered = false
			c.nfProfile.NfStatus = ""
			goto StartNFRegister
		}

		c.doneSignal <- _sigDoneNRFClient
	}
}

// NRF에 SMSF를 등록. 오류 발생 시, 재시도 수행
func (c *NRFClient) registerSMSF() bool {
	retryDuration := time.Duration(c.retryRegisterInterval) * time.Second
	currentRetryDuration := time.Duration(0) * time.Second

	for {
		select {
		case <-time.After(currentRetryDuration):
			if err := c.sendNFRegister(); err != nil {
				loggers.ErrorLogger().Minor("Failed to register SMSF profile: %s", err.Error())
				currentRetryDuration = retryDuration
			} else {
				loggers.InfoLogger().Comment("SMSF profile has been registered to NRF.")
				return true
			}
		case <-c.termSignal:
			loggers.InfoLogger().Comment("NFRegister task has been canceled.")
			return false
		}
	}
}

// NRF에 SMSF 등록 요청 전송
func (c *NRFClient) sendNFRegister() error {
	profile := c.nfProfile
	profile.NfStatus = msg5g.NFStatusRegistered
	//changesSupport := false // 만약을 대비해서 분명하게 의도를 설정
	//profile.NfProfileChangesSupportInd = &changesSupport

	uri := fmt.Sprintf("%s/%s", c.nfInstancesURI, profile.NfInstanceID)
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	loggers.InfoLogger().Comment("sending NFRegister to NRF(%s/%s)", c.nfInstancesURI, profile.NfInstanceID)

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	Resp, err := c.router.SendRequest(context.Background(), uri, "PUT", hdr, profileBytes, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF Registration Fail(uri : %s)", uri)
		return err
	}

	switch Resp.StatusCode {
	case http.StatusCreated: // NFProfile is contained in the response body
		c.retryRegisterCount = 0
		receivedProfile := &msg5g.NFMgmtProfile{}
		err := json.Unmarshal(Resp.ResponseBytes(), receivedProfile)
		if err != nil {
			return err
		}
		/*
			NFProfile 의 nfProfileChangesSupportInd 값(디폴트 값은 false)에 따라
			NFRegister 요청 성공 시의 응답 본문의 컨텐트가 달라진다.
			상기 필드를 true로 설정하여 요청한 경우, 응답 본문에 담기는
			NFProfile 은 필수 애트리뷰트 및 NRF 가 변경한 애트리뷰트, 그리고
			nfProfileChangesInd 애트리뷰트(값은 true)만 추가된다.
			(3GPP TS 29.510, Annex B 참고)

			등록 시에는 실제 NRF에서 처리한 값 그대로 재 설정 하기 위해,
			nfProfileChangesSupportInd 애트리뷰트를 false로 하여 처리 한다.
		*/
		c.nfProfile = receivedProfile
		if receivedProfile.HbTimer != nil {
			newInterval := *receivedProfile.HbTimer
			if c.heartBeatInterval != newInterval {
				loggers.ErrorLogger().Minor("NFHeart-beat interval has been changed from %d to %d by NFRegister", c.heartBeatInterval, newInterval)
				c.heartBeatInterval = newInterval
			}
		}
	default:
		loggers.ErrorLogger().Major("Invalid Registration(retry) : %d, %s",
			Resp.StatusCode, Resp.ResponseBytes())
		c.retryRegisterCount++
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}

	loggers.InfoLogger().Comment("Registration -> RespCode : %d, RespData : %s",
		Resp.StatusCode, string(Resp.ResponseBytes()))
	return nil
}

// NRF에 주기적으로 Heart-beat 전송
func (c *NRFClient) sendNFHeartBeats() error {
	heartBeatDuration := time.Duration(c.heartBeatInterval) * time.Second
	heartBeatFailureCount := 0

	for {
		select {
		case <-time.After(heartBeatDuration):
			if err := c.sendNFHeartBeat(); err != nil {
				loggers.ErrorLogger().Major("Failed to send NFHeart-beat: %s", err.Error())
				heartBeatFailureCount++
				if heartBeatFailureCount >= c.heartBeatFailureThreshold {
					return fmt.Errorf("the number of failures in sending NFHeart-beat has been reached to the threshold: %d", heartBeatFailureCount)
				}
			} else {
				loggers.InfoLogger().Comment("NFHeart-beat has been sent to NRF successfully")
				if c.nfProfile.HbTimer != nil {
					newInterval := *c.nfProfile.HbTimer
					if c.heartBeatInterval != newInterval {
						loggers.InfoLogger().Comment("NFHeart-beat interval has been changed from %d to %d by NFHeart-beat", c.heartBeatInterval, newInterval)
						c.heartBeatInterval = newInterval
						heartBeatDuration = time.Duration(c.heartBeatInterval) * time.Second
					}
				}
			}
		case <-c.termSignal:
			loggers.InfoLogger().Comment("NFHeart-beat task has been canceled")
			return nil
		}
	}
}

// NRF에 Heart-beat 전송
func (c *NRFClient) sendNFHeartBeat() error {
	profile := c.nfProfile
	uri := fmt.Sprintf("%s/%s", c.nfInstancesURI, profile.NfInstanceID)
	/*
		(1) nfStatus 및 load 이외의 추가 정보 전송에 대해서는 추후 상황에 맞춰
			추가 해야 함.
		(2) NFRegister 전송 시와 마찬가지로, nfProfileChangesSupportInd 값을
			false로 전송하면, Response Body에 NFProfile 전체의 내용이 전송
			되도록 함.
			true로 전송하면, NFProfile 의 변경된 사항만 전송되어도 된다는 의미.
			이 경우, Response Body 에는 변경된 애트리뷰트들만 포함되며, 특히
			nfProfileChangesInd 애트리뷰트가 true로 설정되어 포함 됨(3GPP TS
			29.510 Annex B 참고)
			NFHeart-beat 전송은 주기적으로 수행 되므로, 트래픽 오버헤드를 줄이기
			위해 nfProfileChangesSupportInd를 true로 설정하여 요청한다.
	*/
	/*
		patchJSON := `[
			{"op": "replace", "path": "/nfProfileChangesSupportInd", "value": true},
			{"op": "replace", "path": "/nfStatus", "value": "%s"},
			{"op": "replace", "path": "/load", "value": %d}
		]`
	*/

	patchJSON := `[
		{"op": "replace", "path": "/nfStatus", "value": "%s"},
		{"op": "replace", "path": "/load", "value": %d}
	]`
	patchBytes := []byte(fmt.Sprintf(patchJSON, profile.NfStatus, *profile.Load))

	loggers.InfoLogger().Comment("sending NFHeart-beat to NRF(%s)", uri)

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json-patch+json")

	Resp, err := c.router.SendRequest(context.Background(), uri, http.MethodPatch, hdr, patchBytes, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF NFHeart-beat Fail(uri : %s)", uri)
		return err
	}

	switch Resp.StatusCode {
	case http.StatusNoContent:
		loggers.InfoLogger().Comment("Heart-Beat Response : %d", Resp.StatusCode)
		return nil
	case http.StatusOK: // full or partial NFProfile in the body
		receivedProfile := &msg5g.NFMgmtProfile{}
		err := json.Unmarshal(Resp.ResponseBytes(), receivedProfile)
		if err != nil {
			loggers.ErrorLogger().Major("Json Unmarshal Fail")
			return err
		}

		if receivedProfile.NfProfileChangesInd != nil && *receivedProfile.NfProfileChangesInd {
			// Partial NF Profile changes Received
			c.updatePartialNFProfile(Resp.ResponseBytes())
		} else {
			// Full NF Profile received
			c.nfProfile = receivedProfile
		}

		loggers.InfoLogger().Comment("Heart-Beat Response : %d, RespData : %s",
			Resp.StatusCode, string(Resp.ResponseBytes()))
		return nil
	default: // http.StatusNotFound or any other unspecified status in the specification
		loggers.ErrorLogger().Major("Invalid Heart-Beat Response code : %d, body : %s",
			Resp.StatusCode, string(Resp.ResponseBytes()))
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}
}

// NRF에게 SMSF 등록 해지 요청 전송
func (c *NRFClient) sendNFDeregister() error {
	profile := c.nfProfile
	profile.NfStatus = msg5g.NFStatusRegistered

	uri := fmt.Sprintf("%s/%s", c.nfInstancesURI, profile.NfInstanceID)
	loggers.InfoLogger().Comment("sending NFDeregister to NRF(%s)", uri)

	hdr := http.Header{}
	//	hdr.Add("Content-Type", "application/json")
	hdr.Del("Content-Length")
	hdr.Del("Content-Type")

	Resp, err := c.router.SendRequest(context.Background(), uri, http.MethodDelete, hdr, nil, 1*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF DeRegistration Fail(uri : %s)", uri)
		return err
	}

	if Resp.StatusCode != http.StatusNoContent {
		loggers.ErrorLogger().Major("Invalid Status Code : Deregistration %d, %s",
			Resp.StatusCode, string(Resp.ResponseBytes()))
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}
	loggers.InfoLogger().Comment("DeRegistration -> RespCode : %d, RespData : %s",
		Resp.StatusCode, string(Resp.ResponseBytes()))
	return nil
}

//  현재 감시 중인 모든 PCF 상태 감시를 제거
func (c *NRFClient) clearAllMonitoringInfo() {
	pcfInfoTable := c.pcfInfoTable
	idList := pcfInfoTable.GetIDs()
	for _, id := range idList {
		if err := c.sendNFStatusUnsubscribe(id.SubscriptionID); err != nil {
			loggers.ErrorLogger().Major("Failed to unsubscribe PCF (%s) with subscription (%s): %s",
				id.InstanceID, id.SubscriptionID, err.Error())
		} else {
			loggers.InfoLogger().Comment("A subscription (ID: %s) for PCF (%s) has been unsubscribed successfully.", id.SubscriptionID, id.InstanceID)
		}
		pcfInfoTable.Remove(id.InstanceID)
		loggers.InfoLogger().Comment("A PCF (ID: %s) has been removed from the managed table successfully.", id)
	}
}

// 특정 PCF 인스턴스에 대한 NFStatus Notification 구독 요청
func (c *NRFClient) sendNFStatusSubscribe(pcfInfo *PcfInfo) error {
	// Note: ValidityTime 은 BSF에서 정하고 NRF에 요청할 수도 있으나, 결국
	//	Response Body의 validityTime 으로 동작해야 하므로, 요청 시 포함하지 않음
	//TODO: subscriptionData 미리 생성해 놓고 재사용하는 방식으로 추후 수정
	subscriptionData := msg5g.SubscriptionData{
		NfStatusNotificationURI: msg5g.URI(c.nfStatusNotifURI),
		SubscrCond: &msg5g.NfInstanceIDCond{
			NfInstanceID: pcfInfo.InstanceID,
		},
		ReqNotifEvents: []msg5g.NotificationEventType{
			"NF_DEREGISTERED",
		},
	}

	requestBody, err := json.Marshal(subscriptionData)
	if err != nil {
		return err
	}

	loggers.InfoLogger().Comment("sending NFStatusSubscribe to NRF(%s)", c.subscriptionsURI)

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	Resp, err := c.router.SendRequest(context.Background(), c.subscriptionsURI, http.MethodPost, hdr, requestBody, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF NFStatusSubscribe Fail(uri : %s)", c.subscriptionsURI)
		return err
	}
	if Resp.StatusCode != http.StatusNoContent {
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}

	/*
		statusCode, responseBody, err := c.httpRequest(http.MethodPost, c.subscriptionsURI, "application/json", requestBody)
		if err != nil {
			return err
		}

		if statusCode != http.StatusCreated {
			return createProblemDetails(statusCode, responseBody)
		}
	*/
	receivedSubsData := &msg5g.SubscriptionData{}
	if err = json.Unmarshal(Resp.ResponseBytes(), receivedSubsData); err != nil {
		c.sendNFStatusUnsubscribe(pcfInfo.InstanceID)
		return err
	}

	if receivedSubsData.SubscriptionID == nil {
		c.sendNFStatusUnsubscribe(pcfInfo.InstanceID)
		return fmt.Errorf("A required attribute not found: 'subscriptionId'")
	}
	pcfInfo.SubscriptionID = *receivedSubsData.SubscriptionID

	if receivedSubsData.ValidityTime == nil {
		c.sendNFStatusUnsubscribe(pcfInfo.InstanceID)
		return fmt.Errorf("A required attribute not found: 'validityType'")
	}
	pcfInfo.ValidityTime = *receivedSubsData.ValidityTime
	loggers.InfoLogger().Comment("Now pcfInfo.ValidityTime updated to: %s", pcfInfo.ValidityTime.String())
	return nil
}

// 특정 PCF 인스턴스에 대한 NFStatus Notification 구독 해지 요청
func (c *NRFClient) sendNFStatusUnsubscribe(subscriptionID string) error {
	uri := fmt.Sprintf("%s/%s", c.subscriptionsURI, subscriptionID)

	loggers.InfoLogger().Comment("sending NFStatusUnsubscribe to NRF(%s): %s", uri, subscriptionID)

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	Resp, err := c.router.SendRequest(context.Background(), uri, http.MethodDelete, hdr, nil, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF NFStatusUnsubscribe Fail(uri : %s)", uri)
		return err
	}

	if Resp.StatusCode != http.StatusNoContent {
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}

	return nil
}

// NRF에게 지정한 구독 ID에 대한 Validity time 확장 요청
func (c *NRFClient) extendNFStatusNotifyValidityTime(pcfInfo *PcfInfo, newValidityTimeRequest time.Time) error {
	subscriptionID := pcfInfo.SubscriptionID
	uri := fmt.Sprintf("%s/%s", c.subscriptionsURI, subscriptionID)
	patchJSON := `[
		{"op": "replace", "path": "/validityTime", "value": "%s"}
	]`

	newValidityTime := strfmt.DateTime(newValidityTimeRequest)
	patchBytes := []byte(fmt.Sprintf(patchJSON, newValidityTime.String()))

	loggers.InfoLogger().Comment("sending NFStatusSubscriptionUpdate to NRF(%s) with the new validity time: %s", uri, newValidityTimeRequest.String())
	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json-patch+json")

	Resp, err := c.router.SendRequest(context.Background(), uri, http.MethodPatch, hdr, patchBytes, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("NRF NFStatusSubscriptionUpdate Fail(uri : %s)", uri)
		return err
	}

	switch Resp.StatusCode {
	// requested validity time(newValidityTime) accepted by NRF
	case http.StatusNoContent:
		pcfInfo.ValidityTime = newValidityTime
	// SubscriptionData is contained in the response body and it contains new validity time suggested by NRF
	case http.StatusOK:
		receivedSubsData := &msg5g.SubscriptionData{}
		if err = json.Unmarshal(Resp.ResponseBytes(), receivedSubsData); err != nil {
			return err
		}
		if receivedSubsData.ValidityTime == nil {
			return fmt.Errorf("A required attribute not found: 'validityType'")
		}
		pcfInfo.ValidityTime = *receivedSubsData.ValidityTime
	// 400 Bad Reeuqest or 500 Internal Server Error from NRF
	default:
		return createProblemDetails(Resp.StatusCode, Resp.ResponseBytes())
	}

	return nil
}

// NFStatus 구독에 대한 Validity time 체크 수행
func (c *NRFClient) checkNfStatusNotifyValidityTime(pcfInfo *PcfInfo) func() {
	id := pcfInfo.InstanceID

	// 실제 ValidityTime 의 90% 정도의 사긴이 되었을 때, 시간 연장 신청 시도
	currentValidityTime := time.Time(pcfInfo.ValidityTime)
	duration := time.Duration((float64(currentValidityTime.Sub(time.Now()))) * 0.9)
	return func() {
		c.waitGroup.Add(1)
		defer c.waitGroup.Done()

		for {
			select {
			case <-time.After(duration):
				now := time.Now()
				newValidityTimeToRequest := now.Add(c.validityTimeExtention)

				if err := c.extendNFStatusNotifyValidityTime(pcfInfo, newValidityTimeToRequest); err != nil {
					loggers.ErrorLogger().Major("Failed to extend validity time for PCF (ID: %s): %s", id, err.Error())
					now = time.Now()
					/*
						원래의 Validity Time까지 재시도 할 시간적 여유가 남아
						있다면, 3초 간격으로 재시도 수행.
						재시도 할 시간적 여유가 없다면, 결국 expired 될 것이고
						NRF가 자동으로 구독을 해지 할 것 임. 따라서, 구독을 다시
						해야 함. 구독에 성공 했으면, duration 을 업데이트 하고
						통상의 로직을 수행. 실패 했으면 .. 로그 남기고 종료(향후
						개선 필요 할 듯)
					*/
					if now.Add(5 * time.Second).Before(currentValidityTime) { // 5초는 재시도 전송에 대한 확실한 보장을 위해 여유를 둔 것
						// 3초 후 재시도, 현재 설정된 validity time 까지 재 시도
						duration = time.Duration(3) * time.Second
						loggers.InfoLogger().Comment("Retry to extend validity time for PCF (ID: %s) after %s", id, duration)
						break
					} else {
						if err := c.sendNFStatusSubscribe(pcfInfo); err != nil {
							loggers.ErrorLogger().Major("Failed to recover subscription for PCF (ID: %s): %s", id, err.Error())
							return
						}
					}
				}
				now = time.Now()
				newValidityTimeReceived := time.Time(pcfInfo.ValidityTime)
				duration = time.Duration(float64(newValidityTimeReceived.Sub(now)) * 0.9)
				loggers.InfoLogger().Comment("The validity time of PCF (%s) has been extended until %s", id, newValidityTimeReceived)
			case <-pcfInfo.Removing:
				loggers.InfoLogger().Comment("No more validity time checking performed for PCF (ID: %s).", id)
				return
			}
		}
	}
}

/*
// HTTP Request 전송
func (c *NRFClient) httpRequest(httpMethod string, uri string, contentType string, requestBodyBytes []byte) (int, []byte, error) {
	requestBody := bytes.NewBuffer(requestBodyBytes)
	httpRequest, err := http.NewRequest(httpMethod, uri, requestBody)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	httpRequest.Header.Set("Content-Type", contentType)

		var span opentracing.Span
		var copyOfRequest *http.Request
		if c.traceMgr != nil {
			span, copyOfRequest = c.traceMgr.StartSpanFromClientHTTPReq(httpRequest, fmt.Sprintf("%s:/%s", NRFClientServiceName, httpRequest.URL.Path), ulog.InfoLevel)
			if span != nil {
				defer span.Finish()
			}
		}
				httpResponse, err := c.httpClient.Do(httpRequest)
				if err != nil {
					return http.StatusInternalServerError, nil, err
				}
			defer httpResponse.Body.Close()

			if c.traceMgr != nil {
				c.traceMgr.LogHTTPRes(copyOfRequest.Context(), httpResponse, NRFClientServiceName, ulog.InfoLevel)
			}

			responseBody, err := ioutil.ReadAll(httpResponse.Body)
			if err != nil {
				return http.StatusInternalServerError, nil, err
			}

			return httpResponse.StatusCode, responseBody, nil
	return 0, nil, nil
}
*/

// NF Profile 의 일부 애트리뷰트만 변경
func (c *NRFClient) updatePartialNFProfile(profileChangesBytes []byte) error {
	profileChangesMap, err := utils.ConvertJSONBytesToMap(profileChangesBytes)
	if err != nil {
		return err
	}

	nfProfileMap, err := utils.ConvertStructToMap(c.nfProfile)
	if err != nil {
		return err
	}

	changedAttributes := utils.GetMapKeys(profileChangesMap)
	for _, attribName := range changedAttributes {
		nfProfileMap[attribName] = profileChangesMap[attribName]
	}

	updatedProfileBytes, err := json.Marshal(nfProfileMap)
	if err != nil {
		return err
	}

	updatedProfile := &msg5g.NFMgmtProfile{}
	if err = json.Unmarshal(updatedProfileBytes, updatedProfile); err != nil {
		return err
	}
	c.nfProfile = updatedProfile

	return nil
}

func getURIConfig(cfg uconf.Config, configPath string) (string, error) {
	uri := cfg.GetString(configPath, "")
	if len(uri) == 0 {
		return "", fmt.Errorf("An empty URI found")
	}
	return uri, nil
}

func createProblemDetails(statusCode int, responseBody []byte) *msg5g.ProblemDetails {
	problemDetails := &msg5g.ProblemDetails{}
	err := json.Unmarshal(responseBody, problemDetails)
	if err != nil {
		// response body가 ProblemDetails 구조체 형식이 아니라면, 임의로 생성
		problemDetails.Status = statusCode
		problemDetails.Detail = string(responseBody)
	}
	return problemDetails
}
