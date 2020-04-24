package mockups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	scpcli "camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ubsf.git/implements/nbsfsvc"
	"camel.uangel.com/ua5g/ubsf.git/msg5g"
	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/utypes"
	jwt "github.com/dgrijalva/jwt-go"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gin-gonic/gin"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"golang.org/x/net/http2"
)

// NFStatusSubscription NRF NFStatus Subscribe/Unsubscribe 관리용 구조체
type NFStatusSubscription struct {
	subscriptionID string
	nfInstanceID   string
	callbackURI    string
	validityTime   time.Time
}

var logger = ulog.GetLogger("com.uangel.usmsf.nrfsim")

// NRFSimulator NRF Simulator
type NRFSimulator struct {
	testhelper.TestBehavior
	httpClient           *http.Client
	server               *TestServer
	hostAddress          string
	hostName             string
	sharedSecret         string
	accessToken          string
	nfProfiles           map[string]*msg5g.NFMgmtProfile
	subscriptions        map[string]*NFStatusSubscription
	validityTimeDuration time.Duration
	heartBeatCount       int
	termCleanup          chan bool
	started              bool
}

// NewNRFSimulator NRF 시뮬레이터를 생성하고 기동 합니다.
func NewNRFSimulator(cfg uconf.Config) *NRFSimulator {
	hostAddress := cfg.GetString("nrf-sim.host-address", "")
	if len(hostAddress) == 0 {
		logger.Warn("host-address configuration is empty!")
	}
	logger.Debug("Host Address = %s", hostAddress)

	hostName := cfg.GetString("nrf-sim.host-name", "")
	if len(hostName) == 0 {
		logger.Warn("host-name configuration is empty!")
	}
	logger.Debug("Host Name = %s", hostName)

	sharedSecret := cfg.GetString("nrf-sim.shared-secret", "")
	if len(sharedSecret) == 0 {
		logger.Warn("shared-secret configuration is empty!")
	}
	logger.Debug("Shared Secret = %s", sharedSecret)

	d := cfg.GetInt("nrf-sim.validityTimeDuration", 60)
	validityTimeDuration := time.Duration(d) * time.Second

	s := &NRFSimulator{
		httpClient:           &http.Client{},
		hostAddress:          hostAddress,
		hostName:             hostName,
		nfProfiles:           make(map[string]*msg5g.NFMgmtProfile),
		subscriptions:        make(map[string]*NFStatusSubscription),
		sharedSecret:         sharedSecret,
		validityTimeDuration: validityTimeDuration,
		heartBeatCount:       0,
		termCleanup:          make(chan bool),
		started:              false,
	}

	// NRF 시뮬레이터는 기본적으로 1년 간 사용할 수 있는 토큰을 제공
	expirationTime := time.Now().Add(time.Duration(356*24) * time.Hour).Unix()
	token, err := s.createAccessToken(expirationTime)
	if err != nil {
		logger.Error("Failed to create access token: %s", err.Error())
	}
	s.accessToken = token

	tlsConfig := cfg.GetConfig("nrf-sim.tls")
	if tlsConfig != nil {
		ci, err := nbsfsvc.NewCertInfoByCfg(tlsConfig)
		if err != nil {
			logger.With(ulog.Fields{"error": err.Error()}).Error("Failed to create CertInfo instance")
			return nil
		}
		s.httpClient.Transport = &http2.Transport{
			TLSClientConfig: ci.GetClientTLSConfig(),
		}
		s.server, _ = NewTestServerWithHTTP2TLS(hostAddress, hostName, ci.GetServerTLSConfig(), s.handle)
	} else { // HTTP/1.1 사용해야 하는 경우 (현재 구현은 TLS 미사용 하도록 함)
		s.server, _ = NewTestServer(hostAddress, hostName, false, true, s.handle)
		//s.server, _ = NewTestServer(hostAddress, hostName, false, false, s.handle)
		s.httpClient.Transport = &http.Transport{}
	}

	return s
}

// Start NRF 시뮬레이터를 기동 합니다.
func (s *NRFSimulator) Start() {
	logger.Info("Starting NRF Simulator ...")
	s.server.Start()
	go s.cleanupExpiredSubscriptions()
	s.started = true
	logger.Info("NRF Simulator has been started.")
}

// Stop NRF 시뮬레이터를 종료 합니다.
func (s *NRFSimulator) Stop() {
	logger.Warn("Stopping NRF Simulator ...")
	s.termCleanup <- true
	s.server.Stop()
	logger.Info("NRF Simulator has been stopped.")
}

// Close Eager Instance 종료 시 호출 됨
func (s *NRFSimulator) Close() error {
	s.Stop()
	return nil
}

// GetNfProfile 지정한 instance ID 값을 갖는 NF Profile 을 반환. 없으면 nil 반환
func (s *NRFSimulator) GetNfProfile(instanceID string) *msg5g.NFMgmtProfile {
	nfProfile := s.nfProfiles[instanceID]
	return nfProfile
}

// GetHeartBeatCount NFHeart-beat 을 성공적으로 수신한 횟수
func (s *NRFSimulator) GetHeartBeatCount() int {
	return s.heartBeatCount
}

// AddNfProfile NF Profile 을 추가
func (s *NRFSimulator) AddNfProfile(profile *msg5g.NFMgmtProfile) {
	s.nfProfiles[profile.NfInstanceID] = profile
}

// GetNfProfileCount 현재 NRF 시뮬레이터가 관리 중인 NF Profile의 개수
func (s *NRFSimulator) GetNfProfileCount() int {
	return len(s.nfProfiles)
}

// GetNfStatusSubscriberCount 현재 NRF 시뮬레이터의 NFStatus 구독 개수
func (s *NRFSimulator) GetNfStatusSubscriberCount() int {
	return len(s.subscriptions)
}

// SendNfStatusNotify 지정한 nfInstanceID를 기준으로 NFStatusNotify 를 전송
// 전송하는 이벤트는 "NF_DEREGISTERED" 하나만 제공
func (s *NRFSimulator) SendNfStatusNotify(nfInstanceID string) {
	delete(s.nfProfiles, nfInstanceID)

	subscription, ok := s.getSubscription(nfInstanceID)
	if !ok {
		logger.Error("Can't send NFStatusNotify: subscription not found with: %s", nfInstanceID)
		return
	}
	nfInstanceURI := fmt.Sprintf("https://nrf.uangel.com/nnrf-nfm/v1/nf-instances/%s", nfInstanceID)
	notificationData := &msg5g.NotificationData{
		Event:         "NF_DEREGISTERED",
		NfInstanceURI: msg5g.URI(nfInstanceURI),
	}

	requestBody, err := json.Marshal(notificationData)
	if err != nil {
		logger.Error("Can't create response body for NFStatusNotify: %s", err.Error())
		return
	}

	targetURI := subscription.callbackURI
	statusCode, responseBody, err := s.httpRequest(http.MethodPost, targetURI, "application/json", requestBody)
	if err != nil {
		logger.Error("NFStatusNotify request failed: %s", err.Error())
		return
	}

	if statusCode != http.StatusNoContent {
		pd := &msg5g.ProblemDetails{}
		err := json.Unmarshal(responseBody, pd)
		if err != nil {
			logger.Error("Can't parse response body (ProblemDetails): %d %s", statusCode, string(responseBody))
			return
		}

		logger.Error("NRF client responded with an error: %d %s: %s",
			statusCode, pd.Title, pd.Detail)
		return
	}
}

// Started NRF 시뮬레이터가 기동 되었는지 알려준다.
func (s *NRFSimulator) Started() bool {
	return s.started
}

// Handle NRF 시뮬레이터가 수신한 Request를 처리 합니다.
func (s *NRFSimulator) handle(ctx *gin.Context) {
	defer ctx.Request.Body.Close()

	if s.BehaviorEnabled("response-error") {
		s.server.ResponseError(ctx, http.StatusInternalServerError, "intended error", "all requests are ignored")
		return
	}

	fullURIPath := ctx.Request.URL.Path
	requestMethod := ctx.Request.Method
	var uriPath, id string

	switch {
	case strings.Contains(fullURIPath, "oauth"):
		uriPath = fullURIPath
		logger.Debug("URI = %s, Method = %s", uriPath, requestMethod)
	case strings.Contains(fullURIPath, "nf-instances"):
		logger.Debug("FullPath : %s", ctx.Request.URL.RequestURI())
		uriPath, id = s.getURIPathAndID(fullURIPath)
		logger.Debug("URI = %s, nfInstanceID = %s, Method = %s",
			uriPath, id, requestMethod)
	case strings.Contains(fullURIPath, "subscriptions"):
		if requestMethod == "DELETE" || requestMethod == "PATCH" {
			uriPath, id = s.getURIPathAndID(fullURIPath)
			logger.Debug("URI = %s, subscriptionID = %s, Method = %s",
				uriPath, id, requestMethod)
		} else {
			uriPath = fullURIPath
			logger.Debug("URI = %s, Method = %s", uriPath, requestMethod)
		}
	default:
		uriPath = fullURIPath
		id = ""
	}

	switch uriPath {
	case "/oauth2/token": // 규격과 다름에 주의. 단순히 테스트 용 Access Token을 반환 함.
		s.handleOAuth2Token(requestMethod, ctx)
	case "/nnrf-nfm/v1/nf-instances":
		switch requestMethod {
		// NFProfileRetrieval
		case http.MethodGet:
			s.handleGetNfInstances(id, ctx)
		// NFRegister or NFUpdate
		case http.MethodPut:
			s.handlePutNfInstances(id, ctx)
		// NFHeart-beat
		case http.MethodPatch:
			s.handlePatchNfInstances(id, ctx)
		//NFDeregister
		case http.MethodDelete:
			s.handleDeleteNfInstances(id, ctx)
		default:
			s.server.ResponseError(ctx, 404, "Not Found", "not supported operation method: %s", ctx.Request.Method)
		}
	case "/nnrf-nfm/v1/subscriptions":
		switch requestMethod {
		// NFStatusSubscribe
		case http.MethodPost:
			s.handlePostSubscriptions(ctx)
		// NFStatusUnsubscribe
		case http.MethodDelete:
			s.handleDeleteSubscriptions(id, ctx)
		// NFStatusUpdate
		case http.MethodPatch:
			s.handlePatchSubscriptions(id, ctx)
		default:
			s.server.ResponseError(ctx, 404, "Not Found", "not supported operation method: %s", ctx.Request.Method)
		}
	case "/nnrf-disc/v1":
		switch requestMethod {
		//NFDisCovery
		case http.MethodGet:
			s.HandleGetDisCovery(ctx)
		default:
			s.server.ResponseError(ctx, 404, "Not Found", "not supported operation method: %s", ctx.Request.Method)
		}
	default:
		s.server.ResponseError(ctx, 404, "Not Found", "%s%s is not found", ctx.Request.Host, ctx.Request.URL.String())
	}
}

func (s *NRFSimulator) setResponseBodyWithdiscovery(nfProfile scpcli.SearchResult, statusCode int, ctx *gin.Context) {
	// 시뮬레이터에서는 무조건 NF Profile 전체를 전달 해 주도록 한다
	//heartBeatTimer := 3
	//nfProfile.HbTimer = &heartBeatTimer
	//	sendFullNFProfile := false
	//	nfProfile.NfProfileChangesInd = &sendFullNFProfile

	jsonBytes, err := json.Marshal(nfProfile)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error",
			"marshal error: %s", err.Error())
		return
	}
	ctx.Data(statusCode, "application/json", jsonBytes)
}

func (s *NRFSimulator) HandleGetDisCovery(ctx *gin.Context) {

	logger.Info("Discovery : ----> %s\n", ctx.Request.RequestURI)

	/*******************
	INFO[200408 20:44:08.83] Discovery : ----> /nnrf-disc/v1/nf-instances?requester-nf-type=SMSF&target-nf-type=UDM&supi=imsi-450001234567890  file=usmsf/mockups/nrfsim.go line=318 logger=com.uangel.usmsf.nrfsim

	**********************/

	if strings.Contains(ctx.Request.RequestURI, "target-nf-type=UDM") == true {
		nfProfile := s.nfProfiles["045fcaf2-9969-4247-bf19-0c47716a71a9"]

		respBody := scpcli.SearchResult{
			ValidityPeriod: 20,
			NfInstances: []utypes.Map{
				{
					"nfInstanceId":   nfProfile.NfInstanceID,
					"nfType":         nfProfile.NfType,
					"nfStatus":       nfProfile.NfStatus,
					"fqdn":           nfProfile.Fqdn,
					"nfInstanceName": nfProfile.NfType,
					"nfServices":     nfProfile.NfServices,
				}},
			NrfSupportedFeatures: "",
			ExpiresAt:            time.Now().Add(30 * time.Second),
		}

		s.setResponseBodyWithdiscovery(respBody, http.StatusOK, ctx)
		logger.Debug("s.nfProfiles : %v", s.nfProfiles["045fcaf2-9969-4247-bf19-0c47716a71a9"])
	} else {
		nfProfile := s.nfProfiles["030fefe4-06f3-4148-853f-481c30665b22"]

		respBody := scpcli.SearchResult{
			ValidityPeriod: 20,
			NfInstances: []utypes.Map{
				{
					"nfInstanceId":   nfProfile.NfInstanceID,
					"nfType":         nfProfile.NfType,
					"nfStatus":       nfProfile.NfStatus,
					"fqdn":           nfProfile.Fqdn,
					"nfInstanceName": nfProfile.NfType,
					"nfServices":     nfProfile.NfServices,
				}},
			NrfSupportedFeatures: "",
			ExpiresAt:            time.Now().Add(30 * time.Second),
		}

		s.setResponseBodyWithdiscovery(respBody, http.StatusOK, ctx)
		logger.Debug("s.nfProfiles : %v", s.nfProfiles["030fefe4-06f3-4148-853f-481c30665b22"])

	}

}

func (s *NRFSimulator) cleanupExpiredSubscriptions() {
	for {
		select {
		case <-time.After(1 * time.Second):
			now := time.Now()
			expiredSubscriptions := []string{}

			mutex.RLock()
			for subsID, subscription := range s.subscriptions {
				mutex.RUnlock()
				if subscription.validityTime.Before(now) {
					logger.Debug("subscription.validityTime = %s | checking time = %s", subscription.validityTime.String(), now.String())

					expiredSubscriptions = append(expiredSubscriptions, subsID)
					logger.Info("subscription expired: %s", subsID)
				}
				mutex.RLock()
			}
			mutex.RUnlock()
			for _, subsID := range expiredSubscriptions {
				mutex.Lock()
				delete(s.subscriptions, subsID)
				mutex.Unlock()
				logger.Debug("subscription removed: %s, subscriptions: %d", subsID, len(s.subscriptions))
			}
		case <-s.termCleanup:
			logger.Info("Cleaning-up expired subscriptions task has been canceled.")
			return
		}
	}
}

func (s *NRFSimulator) createAccessToken(expirationTime int64) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["iss"] = s.hostName
	claims["sub"] = "self" // NF service consumer's NF instance ID
	claims["aud"] = "BSF"  // NFType or array of NF service producer's NF instance IDs
	claims["scope"] = "nbsf-management"
	claims["exp"] = expirationTime

	return token.SignedString([]byte(s.sharedSecret))
}

func (s *NRFSimulator) handleOAuth2Token(httpMethod string, ctx *gin.Context) {
	if httpMethod == http.MethodPost {
		// Request body check는 생략. BSF 와의 API 연동에 필요한 기능만 제공
		responseBody := `{
			"access_token": "%s"
		}`

		if s.BehaviorEnabled("access-token-expires-in-3sec") {
			expirationTime := time.Now().Add(3 * time.Second).Unix()
			accessToken, err := s.createAccessToken(expirationTime)
			if err != nil {
				message := fmt.Sprintf("Failed to create access token: %s", err.Error())
				logger.Error(message)
				s.server.ResponseError(ctx, http.StatusInternalServerError,
					"Internal Server Error", message)
				return
			}
			responseBody = fmt.Sprintf(responseBody, accessToken)
			logger.Warn("access token will be expired in 5 seconds")
		} else {
			responseBody = fmt.Sprintf(responseBody, s.accessToken)
		}
		ctx.Data(http.StatusOK, "application/json", []byte(responseBody))
	} else {
		s.server.ResponseError(ctx, http.StatusNotFound, "Not Found",
			"not supported operation method: %s", ctx.Request.Method)
	}
}

func (s *NRFSimulator) handleGetNfInstances(nfInstanceID string, ctx *gin.Context) {
	if nfProfile, ok := s.nfProfiles[nfInstanceID]; ok {
		s.setResponseBodyWithFullNfProfile(nfProfile, http.StatusCreated, ctx)
	} else {
		s.server.ResponseError(ctx, http.StatusNotFound,
			"Not Found", "NF instance ID not found: %s", nfInstanceID)
	}
}

/* NFRegister 및 NFUpdate 는 동일한 API 사용:
	존재하지 않는 nfInstanceID 인 경우, NRF 는 Register 수행
 	존재하는 nfInstanceID 인 경우, NRF 는 Update 수행
*/
func (s *NRFSimulator) handlePutNfInstances(nfInstanceID string, ctx *gin.Context) {
	if nfProfile := s.getNfProfileFromRequestBody(ctx); nfProfile != nil {
		/* 현재 NRF 시뮬레이터는 NFRegister 및 NFUpdate 에 대하여 Full Replace
		   방식으로만 동작하고 있음.
		   규격에 의하면, NFRegister 시에는 NRF가 요구하는 Heart-beat Interval을
		   반환하는 NFProfile에 설정하여 넘겨 주도록 되어 있음에 주의.
		*/
		s.nfProfiles[nfInstanceID] = nfProfile
		s.setResponseBodyWithFullNfProfile(nfProfile, http.StatusCreated, ctx)
	} else {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "Request body is empty")
	}
}

func (s *NRFSimulator) handlePatchNfInstances(nfInstanceID string, ctx *gin.Context) {
	requestBody := s.getRequestBody(ctx)

	if modifiedProfile, err := s.patchNfProfile(nfInstanceID, requestBody); err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "Error in patching Heart-beat request")
	} else {
		s.nfProfiles[nfInstanceID] = modifiedProfile
		s.setResponseBodyWithFullNfProfile(modifiedProfile, http.StatusOK, ctx)
		s.heartBeatCount++
		logger.Debug("NFHeart-beat received from %s", nfInstanceID)
	}
}

func (s *NRFSimulator) handleDeleteNfInstances(nfInstanceID string, ctx *gin.Context) {
	if _, ok := s.nfProfiles[nfInstanceID]; ok {
		//delete(s.nfProfiles, nfInstanceID)
		ctx.Data(http.StatusNoContent, "application/json", nil)
		s.SendNfStatusNotify(nfInstanceID)
	} else {
		s.server.ResponseError(ctx, 404, "Not Found", "the requested instance doesn't exist: %s", nfInstanceID)
	}
}

func (s *NRFSimulator) handlePostSubscriptions(ctx *gin.Context) {
	subscriptionData := &msg5g.SubscriptionData{}
	requestBody := s.getRequestBody(ctx)
	if err := json.Unmarshal(requestBody, subscriptionData); err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "can't parse request body: %s", err.Error())
		return
	}

	subscription, err := s.addSubscription(subscriptionData)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "cant' create a subscription: %s", err.Error())
		return
	}

	subscriptionData.SubscriptionID = &subscription.subscriptionID

	var validityTime strfmt.DateTime
	if s.BehaviorEnabled("fixed-validity-time") {
		validityTime = strfmt.DateTime(time.Now().Add(5 * time.Second))
	} else {
		validityTime = strfmt.DateTime(subscription.validityTime)
	}
	subscriptionData.ValidityTime = &validityTime

	responseBytes, err := json.Marshal(subscriptionData)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "can't create response body: %s", err.Error())
		return
	}

	scheme := "https"
	resourcePath := fmt.Sprintf("%s://%s%s/%s",
		scheme, ctx.Request.Host, ctx.Request.RequestURI,
		subscription.subscriptionID)
	ctx.Header("Location", resourcePath)
	ctx.Data(http.StatusCreated, "application/json", responseBytes)

	logger.Info("NF (%s) subscribed to NFStatus successfully: %s",
		subscription.nfInstanceID, resourcePath)
}

func (s *NRFSimulator) handleDeleteSubscriptions(subscriptionID string, ctx *gin.Context) {
	nfInstanceID, ok := s.removeSubscription(subscriptionID)
	if !ok {
		s.server.ResponseError(ctx, http.StatusNotFound, "Not Found",
			"requested subscription was not found: %s", subscriptionID)
		return
	}
	ctx.Data(http.StatusNoContent, "application/json", nil)

	logger.Info("NF (%s) unsubscribed from NFStatus successfully: %s",
		nfInstanceID, subscriptionID)
}

func (s *NRFSimulator) handlePatchSubscriptions(subscriptionID string, ctx *gin.Context) {
	requestBody := s.getRequestBody(ctx)
	subscriptionData, err := s.patchSubscriptionData(subscriptionID, requestBody)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusBadRequest, "Bad Request",
			"Can't parse the request body: %s", err.Error())
		return
	}

	mutex.RLock()
	subscription := s.subscriptions[subscriptionID]
	mutex.RUnlock()
	/*
		기본적으로 NRF 시뮬레이터는 NF 가 요청한 연장 시간을 그대로 허용 하도록
		하지만, 테스트를 위해 Behavior를 지정한 경우에는 그 설정에 따른다.
	*/
	if s.BehaviorEnabled("fixed-validity-time") {
		// 5초만 연장 하도록 고정
		subscription.validityTime = time.Now().Add(5 * time.Second)
	} else {
		subscription.validityTime = time.Time(*subscriptionData.ValidityTime)
	}
	vt := strfmt.DateTime(subscription.validityTime)
	subscriptionData.ValidityTime = &vt

	responseBody, err := json.Marshal(subscriptionData)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error", "can't create response body: %s", err.Error())
		return
	}

	/*
		규격에서는 NF 가 요청한 연장 시간을 수용 가능한 경우, 그 시간을
		사용해야 하고, 이 경우에는 Response Body 없이 "204 No Content" 상태
		코드를 전달 해야 함.
	*/
	ctx.Data(http.StatusOK, "application/json", responseBody)
}

func (s *NRFSimulator) getURIPathAndID(fullURIPath string) (uriPath string, id string) {
	from := strings.LastIndex(fullURIPath, "/")
	id = fullURIPath[from+1:]
	uriPath = fullURIPath[:from]

	return uriPath, id
}

func (s *NRFSimulator) getNfProfileFromRequestBody(ctx *gin.Context) *msg5g.NFMgmtProfile {
	nfProfile := &msg5g.NFMgmtProfile{}

	requestBody := s.getRequestBody(ctx)
	err := json.Unmarshal(requestBody, nfProfile)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusBadRequest,
			"Bad Request",
			"parse error: %s", err.Error())
		return nil
	}

	return nfProfile
}

func (s *NRFSimulator) setResponseBodyWithFullNfProfile(nfProfile *msg5g.NFMgmtProfile, statusCode int, ctx *gin.Context) {
	// 시뮬레이터에서는 무조건 NF Profile 전체를 전달 해 주도록 한다
	//heartBeatTimer := 3
	//nfProfile.HbTimer = &heartBeatTimer
	sendFullNFProfile := false
	nfProfile.NfProfileChangesInd = &sendFullNFProfile

	jsonBytes, err := json.Marshal(nfProfile)
	if err != nil {
		s.server.ResponseError(ctx, http.StatusInternalServerError,
			"Internal Server Error",
			"marshal error: %s", err.Error())
		return
	}
	ctx.Data(statusCode, "application/json", jsonBytes)
}

func (s *NRFSimulator) patchNfProfile(nfInstanceID string, patchJSON []byte) (*msg5g.NFMgmtProfile, error) {
	if len(s.nfProfiles) == 0 {
		message := fmt.Sprintf("It's not registered NF instance: %s", nfInstanceID)
		logger.Error(message)
		return nil, fmt.Errorf(message)
	}

	profileJSON, err := json.Marshal(s.nfProfiles[nfInstanceID])
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, err
	}

	modifiedJSON, err := patch.Apply(profileJSON)
	if err != nil {
		return nil, err
	}

	modifiedProfile := &msg5g.NFMgmtProfile{}
	if err := json.Unmarshal(modifiedJSON, modifiedProfile); err != nil {
		return nil, err
	}

	return modifiedProfile, nil
}

func (s *NRFSimulator) patchSubscriptionData(subscriptionID string, patchJSON []byte) (*msg5g.SubscriptionData, error) {
	mutex.RLock()
	subscription, ok := s.subscriptions[subscriptionID]
	mutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("Subscription not found: %s", subscriptionID)
	}

	var validityTime strfmt.DateTime
	validityTime = strfmt.DateTime(subscription.validityTime)
	subscriptionData := &msg5g.SubscriptionData{
		ValidityTime: &validityTime,
	}

	subsDataJSON, err := json.Marshal(subscriptionData)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, err
	}

	modifiedJSON, err := patch.Apply(subsDataJSON)
	if err != nil {
		return nil, err
	}

	modifiedSubsData := &msg5g.SubscriptionData{}
	if err := json.Unmarshal(modifiedJSON, modifiedSubsData); err != nil {
		return nil, err
	}

	return modifiedSubsData, nil
}

func (s *NRFSimulator) getRequestBody(ctx *gin.Context) []byte {
	var requestBodyBytes []byte
	requestBodyBytes, _ = ioutil.ReadAll(ctx.Request.Body)
	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(requestBodyBytes))

	return requestBodyBytes
}

/* [제한 사항]
   SubscriptionData에 대하여:
   - SubscrCond 는 NNfInstanceIDCond 만 허용
   - ReqNotiEvents 는 NotificationEventType (열거형) 중 "NF_DEREGISTERED" 만
	 허용
*/
var mutex = &sync.RWMutex{}

func (s *NRFSimulator) addSubscription(subscriptionData *msg5g.SubscriptionData) (*NFStatusSubscription, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	subsID := uuid.String()
	validityTime := time.Now().Add(s.validityTimeDuration)

	newSubscription := &NFStatusSubscription{
		nfInstanceID:   subscriptionData.SubscrCond.NfInstanceID,
		subscriptionID: subsID,
		callbackURI:    string(subscriptionData.NfStatusNotificationURI),
		validityTime:   validityTime,
	}
	mutex.Lock()
	s.subscriptions[subsID] = newSubscription
	mutex.Unlock()
	return newSubscription, nil
}

func (s *NRFSimulator) getSubscription(nfInstanceID string) (*NFStatusSubscription, bool) {
	mutex.RLock()
	for _, subs := range s.subscriptions {
		mutex.RUnlock()
		if subs.nfInstanceID == nfInstanceID {
			return subs, true
		}
		mutex.RLock()
	}
	mutex.RUnlock()
	return nil, false
}

func (s *NRFSimulator) removeSubscription(subscriptionID string) (nfInstanceID string, ok bool) {
	mutex.RLock()
	subscription, ok := s.subscriptions[subscriptionID]
	mutex.RUnlock()
	if ok {
		nfInstanceID = subscription.nfInstanceID
		mutex.Lock()
		delete(s.subscriptions, subscriptionID)
		mutex.Unlock()
	}
	return
}

// HTTP Request 전송
func (s *NRFSimulator) httpRequest(httpMethod string, uri string, contentType string, requestBodyBytes []byte) (int, []byte, error) {
	requestBody := bytes.NewBuffer(requestBodyBytes)
	httpRequest, err := http.NewRequest(httpMethod, uri, requestBody)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	httpRequest.Header.Set("Content-Type", contentType)
	httpResponse, err := s.httpClient.Do(httpRequest)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	defer httpResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return httpResponse.StatusCode, responseBody, nil
}
