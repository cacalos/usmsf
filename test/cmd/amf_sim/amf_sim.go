package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os/signal"

	"os"

	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	jsoniter "github.com/json-iterator/go"
	"github.com/philippfranke/multipart-related/related"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/hocon"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"

	cpdu "camel.uangel.com/ua5g/usmsf.git/endecoder/cpdu"
	rpdu "camel.uangel.com/ua5g/usmsf.git/endecoder/rpdu"

	"camel.uangel.com/ua5g/usmsf.git/common"

	"encoding/json"
	"net/textproto"
	"runtime"

	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

//HTTPServer HTTP 기반 서비스 서버의 공통 객체
type HTTPServer struct {
	Addr               string
	Handler            *gin.Engine
	authRequired       bool     //Authenticiation이 요구 되는지 여부
	authCredentials    [][]byte // slice with base64-encoded credentials
	probeResistDomain  string
	probeResistEnabled bool
}

// RespondProblemDetails 전달된 Problem Details 에러를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondProblemDetails(ctx *gin.Context, pd *msg5g.ProblemDetails) error {
	rbody, err := json.Marshal(pd)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
	} else {
		ctx.Data(pd.Status, "application/problem+json", rbody)
	}
	return pd
}

// RespondSystemError System Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondSystemError(ctx *gin.Context, err error) error {
	pd := msg5g.SystemError(ctx.Request.URL.String(), err)
	return s.RespondProblemDetails(ctx, pd)
}

// Errorf 전달된 이름의 Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) Errorf(code int, title string, formatstr string, args ...interface{}) *msg5g.ProblemDetails {
	err := fmt.Errorf(formatstr, args...)
	pd := &msg5g.ProblemDetails{
		Title:  title,
		Status: code,
		Detail: err.Error(),
	}
	return pd
}

// ErrorNotFound 전달된 Resource를 찾을 수 없다.
func (s *HTTPServer) ErrorNotFound(ctx *gin.Context) *msg5g.ProblemDetails {
	return s.Errorf(404, "Not Found", "'%v' is not found.", ctx.Request.URL.String())
}

// ErrorBadRequest 전달된 Request가 문제가 있다.
func (s *HTTPServer) ErrorBadRequest(ctx *gin.Context) *msg5g.ProblemDetails {
	return s.Errorf(400, "Bad Request", "'%v' is bad request.", ctx.Request.URL.String())
}

// ErrorInvalidProtocol 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorInvalidProtocol(ctx *gin.Context) *msg5g.ProblemDetails {
	return s.Errorf(400, "Bad Request", "'%v %v' is sent on invalid protocol.", ctx.Request.Method, ctx.Request.URL.String())
}

// ErrorUnauthorized 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorUnauthorized(ctx *gin.Context, sender, message string) *msg5g.ProblemDetails {
	return s.Errorf(401, "Unauthorized", "'%v' is unauthorized.%v", sender, message)
}

// ErrorForbidden 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorForbidden(ctx *gin.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(403, "Forbidden", "'%v'.%v", ctx.Request.URL.String(), message)
}

// ErrorNotAllowed 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorNotAllowed(ctx *gin.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(405, "NotAllowed", "'%v'.%v", ctx.Request.URL.String(), message)
}

// ErrorServiceUnavailable 서비스를 이용할 수 없다.
func (s *HTTPServer) ErrorServiceUnavailable(ctx *gin.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(503, "ServiceUnavailable", "'%v'.%v", ctx.Request.URL.String(), message)
}

var idx uint32

func isContain(strs []string, str string) bool {
	for i := range strs {
		if strs[i] == str {
			return true
		}
	}
	return false
}

type Config struct {
	cc        int
	operation string
}

type AmfServer struct {
	HTTPServer

	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server
	smsClient   []*common.HTTPClient
	conf        Config

	r_metrics *HTTPMetrics
	s_metrics *HTTPMetrics

	connCnt  uint32
	failNoti bool
	cause    string

	httpsAddr string
}

const IfServerSvcName = "AMFSIM"

var verbose bool

func NewAmfServer(cfg uconf.Config) *AmfServer {

	var cliConf common.HTTPCliConf
	var err error

	httpConf := cfg.GetConfig("amf-sim.http")
	httpsConf := cfg.GetConfig("amf-sim.https")
	smsfConf := cfg.GetConfig("amfsim")

	s := &AmfServer{}
	//	smsfhost := os.Getenv("SMSF_HOST")

	if smsfConf == nil {
		fmt.Printf("amfsim Parsing Fail")
		return nil
	}

	smsfhost := smsfConf.GetString("smsf-host", "")
	cliConf.DialTimeout = smsfConf.GetDuration("map-client.connection.timeout", time.Second*20)
	cliConf.DialKeepAlive = smsfConf.GetDuration("map-client.connection.keep-alive", time.Second*20)
	cliConf.IdleConnTimeout = smsfConf.GetDuration("map-client.connection.expire-time", 1*time.Minute)
	cliConf.InsecureSkipVerify = true
	s.connCnt = uint32(smsfConf.GetInt("client-conn", 5))
	if smsfhost != "" {
		s.smsClient = make([]*common.HTTPClient, s.connCnt, s.connCnt)

		var i uint32

		for i = 0; i < s.connCnt; i++ {
			s.smsClient[i], err = common.NewHTTPClient(&cliConf, "http", smsfhost, smsfhost, 2, nil)
			if err != nil {
				fmt.Println("Failed to create SMS Client")
				return nil
			}
		}

	}

	s.conf.cc = smsfConf.GetInt("cc", 0)
	s.conf.operation = smsfConf.GetString("operation", "")

	s.failNoti = smsfConf.GetBoolean("failure-noti", false)
	s.cause = smsfConf.GetString("failure-noti-cause", "UE_NOT_RESPONDING")
	verbose = smsfConf.GetBoolean("verbose", false)

	s.Handler = gin.New()

	s.r_metrics = NewHTTPMetrics()
	s.s_metrics = NewHTTPMetrics()

	s.http2SvrCfg = &http2.Server{
		MaxHandlers:          cfg.GetInt("amf-sim.https.max-handler", 0),
		MaxConcurrentStreams: uint32(cfg.GetInt("amf-sim.https.max-concurrent-streams", 20000)),
	}

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		httpPort := httpConf.GetInt("port", 8085)
		s.Addr = httpAddr + ":" + strconv.Itoa(httpPort)
		httpsvr := &http.Server{
			Addr:    s.Addr,
			Handler: h2c.NewHandler(s.Handler, s.http2SvrCfg),
		}
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	}

	if httpsConf != nil {
		httpsAddr := httpsConf.GetString("address", "")
		httpsPort := httpsConf.GetInt("port", 9095)
		tlscfg := cfg.GetConfig("amf-sim.tls")
		if tlscfg == nil {
			tlscfg = cfg.GetConfig("amf.tls.internal-network")
			if tlscfg == nil {
				tlscfg = cfg.GetConfig("amf.tls")
				if tlscfg == nil {
					fmt.Printf("Not found TLS configuration (als-server.tls| sepp-tls.internal-ntework| sepp.tls)")
					return nil
				}
			}
		}

		certInfo, err := common.NewCertInfoByCfg(tlscfg)
		if err != nil {
			fmt.Println("Failed to create CertInfo")
			return nil
		}

		s.httpsAddr = httpsAddr + ":" + strconv.Itoa(httpsPort)
		httpssvr := &http.Server{
			Addr:      s.httpsAddr,
			Handler:   s.Handler,
			TLSConfig: certInfo.GetServerTLSConfig(),
		}
		http2.ConfigureServer(httpssvr, s.http2SvrCfg)
		s.httpsServer = httpssvr
	}

	go s.ReportStat()

	s.Handler.PUT("/:s/:v/:o/:n/:m", s.Handle)
	s.Handler.GET("/:s/:v/:n/:o", s.Handle)
	s.Handler.POST("/:s/:v/:o/:n/:m", s.Handle)

	return s
}

func (s *AmfServer) ReportStat() {

	reportPeriod := 1
	start := time.Now()
	reportSec := start.Unix()

	for {
		now := time.Now()
		nowSecond := now.Unix()

		if int(nowSecond-reportSec) >= reportPeriod {
			reportSec = nowSecond
			fmt.Printf("[%v]\n", time.Now())
			fmt.Printf("[Receive Message ]\n")
			s.r_metrics.R_Report(true, true, false)
			fmt.Printf("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - \n")
			fmt.Printf("[Send Message    ]\n")
			s.s_metrics.S_Report(true, true, false)
			fmt.Printf("\n")
		}

		time.Sleep(1 * time.Second)
	}
}

func (s *AmfServer) Start() {
	if s.httpServer != nil {
		go func() {
			err := s.httpServer.ListenAndServe()
			if err != nil {
				fmt.Printf("Shutting down the AMFSIM http://%v\n", s.Addr)
			} else {
				fmt.Printf("Failed to listen and serve AMFSIM http://%v\n", s.Addr)
			}
		}()
	}

	if s.httpsServer != nil {
		go func() {
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				fmt.Printf("Shutting down the AMFSIM https://%v\n", s.httpsAddr)
			} else {
				fmt.Printf("Failed to listen and serve AMFSIM https://%v\n", s.httpsAddr)
			}
		}()
	}

}

func (s *AmfServer) Handle(ctx *gin.Context) {

	var err error

	service := ctx.Param("s")
	version := ctx.Param("v")
	supi := ctx.Param("n")
	operation := ctx.Param("o")
	msgType := ctx.Param("m")

	if ctx.Request.Body != nil {
		defer ctx.Request.Body.Close()
	}

	switch service {
	case "namf-comm":
		if version != "v1" || len(supi) == 0 {
			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

		}

		if ctx.Request.Method == "POST" {
			if operation == "ue-contexts" {
				if msgType == "n1-n2-messages" {
					s.handleN1N2Message(ctx, supi)
				} else {
					fmt.Printf("Unsupported Request msgType : %s\n", msgType)
					err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
				}

			} else {
				fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
			}
		} else {
			fmt.Printf("Unsupported Request Method : %s\n", ctx.Request.Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

		}

	case "namf-mt":
		if version != "v1" || len(supi) == 0 {
			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
		}

		switch ctx.Request.Method {
		case "PUT":
			if operation == "ue-contexts" {
				if msgType == "ue-reachind" {
					s.handleReach(ctx, supi)
				} else {
					fmt.Printf("Unsupported Request Operation : %s\n", operation)
					err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
				}
			} else {
				fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

			}

		default:
			fmt.Printf("Unsupported Request Method : %s\n", ctx.Request.Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
		}

	default:
		fmt.Printf("Unsupported Service : %s\n", service)
		err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

	}

	if err != nil {
		fmt.Println(err.Error())
	}

	return
}

func (s *AmfServer) handleN1N2Message(c *gin.Context, supi string) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var jsonContentId string
	var bodyOfBinaryPart []byte
	var msgRef byte
	var msgType byte

	now := s.r_metrics.Start()

	if verbose == true {
		fmt.Println("< AMF(N1N2) Service > -----------------------------------------------------------------------")
		fmt.Printf("Recevie POST N1N2 Msg : User - %s\n", supi)
	}

	user := supi
	contentsType := c.Request.Header.Get("Content-Type")
	n1n2Msg := msg5g.N1N2Request{}

	mediaType, params, err := mime.ParseMediaType(contentsType)

	messageBody, err := c.GetRawData()
	if err != nil {
		fmt.Printf("err : %s\n", err)
		_ = s.RespondSystemError(c, errcode.BadRequest(c.Request.URL.Path))
		s.r_metrics.N1n2Stop(now, http.StatusBadRequest)
		return
	}

	if strings.Compare(mediaType, "multipart/related") == 0 {
		bytesBuf := bytes.NewReader(messageBody)
		r := related.NewReader(bytesBuf, params)

		part, err := r.NextPart()
		for part != nil && err == nil {
			if part.Root {
				if isContain(part.Header["Content-Type"], "application/json") {
					bodyOfJSONPart, err := ioutil.ReadAll(part)
					if err != nil {
						fmt.Println("Read Error JSON Data")
						_ = s.RespondSystemError(c, errcode.BadRequest(c.Request.URL.Path))
						s.r_metrics.N1n2Stop(now, http.StatusBadRequest)
						return
					}
					if verbose == true {
						fmt.Printf("JSON contents : %s\n", bodyOfJSONPart)
					}
					err = json.Unmarshal(bodyOfJSONPart, &n1n2Msg)
					if err != nil {
						fmt.Printf("JSON unmarshalling Error, err:%s\n", err)
						_ = s.RespondSystemError(c, errcode.BadRequest(c.Request.URL.Path))
						s.r_metrics.N1n2Stop(now, http.StatusBadRequest)
						return
					}
					jsonContentId = n1n2Msg.N1MessageContainer.N1MessageContent.ContentID
					if verbose == true {
						fmt.Printf("contentId:%s\n", jsonContentId)
					}
				} else {
					fmt.Printf("Fail to parse Content-Type of the root part : %s, it should be application/json, USER:%s\n", contentsType, user)

					c.Header("Content-Type", "application/problem+json")
					c.JSON(http.StatusForbidden, gin.H{"cause": "SERVICE_NOT_ALLOWED"})
					s.r_metrics.N1n2Stop(now, http.StatusForbidden)
					return
				}
			} else {
				if isContain(part.Header["Content-Type"], "application/vnd.3gpp.5gnas") {

					contentsId := part.Header["Content-Id"][0][0:len(part.Header["Content-Id"][0])]
					if len(contentsId) == 0 {
						fmt.Printf("Does not exist Content-Id, USER:%s\n", user)
						c.Header("Content-Type", "application/problem+json")
						c.JSON(http.StatusNotFound, gin.H{"cause": "CONTEXT_NOT_FOUND"})
						s.r_metrics.N1n2Stop(now, http.StatusNotFound)
						return
					} else if contentsId != jsonContentId {
						fmt.Printf("binary data header contentId : %s, JSON data contentId : %s\n", contentsId, jsonContentId)
						c.Header("Content-Type", "application/problem+json")
						c.JSON(http.StatusNotFound, gin.H{"cause": "CONTEXT_NOT_FOUND"})
						s.r_metrics.N1n2Stop(now, http.StatusNotFound)
						return
					}

					bodyOfBinaryPart, err = ioutil.ReadAll(part) //binary

					if err != nil {
						fmt.Printf("Fail to read body of binary part : %s\n", err)
						c.Header("Content-Type", "application/problem+json")
						c.JSON(http.StatusBadRequest, gin.H{"cause": "SMS_PAYLOAD_ERROR"})
						s.r_metrics.N1n2Stop(now, http.StatusBadRequest)
						return
					}
					if verbose == true {
						fmt.Printf("Content-ID : %s\n", contentsId)
						fmt.Printf("contents : %x\n", bodyOfBinaryPart)
					}
					if bodyOfBinaryPart[1] == 0x01 {
						cpdata := cpdu.Decoding(bodyOfBinaryPart)
						if verbose == true {
							fmt.Printf("MSG is CP-DATA, value = %x\n", bodyOfBinaryPart[1])
							fmt.Printf("RP-DATA, value = %x\n", cpdata.CpUserData)
						}
						rpdata := rpdu.Decoding(cpdata.CpUserData)
						msgRef = rpdata.RpMessageReference
						msgType = rpdata.RpMessageType

					} else if bodyOfBinaryPart[1] == 0x04 {
						if verbose == true {
							fmt.Printf("MSG is CP-ACK, value = %x\n", bodyOfBinaryPart[1])
						}
					} else {
						if verbose == true {
							fmt.Printf("MSG is CP-ERROR, value = %x\n", bodyOfBinaryPart[1])
						}
					}
				} else {
					fmt.Printf("Fail to parse Content-Type of the part : %s, it should be application/3gpp.vnd.com\n", contentsType)
					c.Header("Content-Type", "application/problem+json")
					c.JSON(http.StatusBadRequest, gin.H{"cause": "SMS_PAYLOAD_MISSING"})
					s.r_metrics.N1n2Stop(now, http.StatusBadRequest)
					return
				}

			}
			part, err = r.NextPart()
		}

	} else if strings.Compare(mediaType, "application/json") != 0 {

	}

	c.String(http.StatusOK, "POST N1N2 Transfer OK")

	if s.failNoti == true || bodyOfBinaryPart[1] == 0x01 {
		val := atomic.AddUint32(&idx, 1)
		cli := s.smsClient[val%s.connCnt]
		if s.failNoti == true {
			go s.FailureNoti(user, s.cause, n1n2Msg.N1n2FailureTxfNotifURI, cli)
		} else if bodyOfBinaryPart[1] == 0x01 {
			go s.SendUplinkMsg(user, 0, 0, s.conf.cc, cli)
			if msgType == rpdu.RP_DATA_N_MS {
				val = atomic.AddUint32(&idx, 1)
				cli = s.smsClient[val%s.connCnt]
				go s.SendUplinkMsg(user, 1, msgRef, s.conf.cc, cli)
			}
		}
	}

	if verbose == true {
		fmt.Println("< END(N1N2) Service > --------------------------------------------------------------------")
	}

	s.r_metrics.N1n2Stop(now, http.StatusOK)
}

func (s *AmfServer) handleReach(c *gin.Context, supi string) {
	var cc, code int
	var operation string

	now := s.r_metrics.Start()

	cc = s.conf.cc
	operation = s.conf.operation
	user := supi

	if verbose == true {
		fmt.Println("< AMF(REACH) Service > -----------------------------------------------------------------------")
		fmt.Printf("Recevie PUT UeReachibiltyEnableReq Msg : User - %s\n", user)
	}

	if operation == "amf" && cc > 0 {
		c.Header("Content-Type", "application/problem+json")
		c.String(cc, "Unable Status for Deliver MT MSG")
		if verbose == true {
			fmt.Printf("Nack Error : %d, User - %s\n", cc, user)
		}
		code = cc
	} else {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, "Enable Status for Deliver MT MSG")
		code = http.StatusOK
	}

	if verbose == true {
		fmt.Println("< END(REACH) Service > --------------------------------------------------------------------")
	}

	s.r_metrics.ReachStop(now, code)

	return
}

func (s *AmfServer) FailureNoti(supi string, cause string, notiUri string, client *common.HTTPClient) {

	now := s.s_metrics.Start()

	if verbose == true {
		fmt.Println("< AMF(FAILURE NOTI SEND) > ---------------------------------------------------------------------")
	}
	smsfURL := fmt.Sprintf("/namf-svc/v1/sms-failure-notify/%s", supi)

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	reqBody := MakeFailNotiMsg(cause, notiUri)
	uplinkResp, respData, err := client.Call("POST", smsfURL, hdr, reqBody)

	if err != nil {
		fmt.Printf("Send Fail Uplink Msg, SUPI:%s\n", supi)
		s.s_metrics.FailNotiStop(now, uplinkResp.StatusCode)
		return
	}

	if verbose == true {
		fmt.Printf("resp code : %d, data : %s\n", uplinkResp.StatusCode, string(respData))
		fmt.Println("< END(UPLINK SEND) Service > -----------------------------------------------------------------------")
	}

	s.s_metrics.FailNotiStop(now, uplinkResp.StatusCode)
}

func (s *AmfServer) SendUplinkMsg(supi string, cptype int, msgRef byte, cc int, client *common.HTTPClient) {

	now := s.s_metrics.Start()

	if verbose == true {
		fmt.Println("< AMF(UPLINK SEND) > -----------------------------------------------------------------------")
	}
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS8iLCJhdWQiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS9uc21zZi92Mi80NTAwNjEyMzQ1Njc4Iiwic3ViIjoidXNyXzEyMyIsInNjb3BlIjoicmVhZCB3cml0ZSIsImlhdCI6MTQ1ODc4NTc5NiwiZXhwIjoxNjY4ODcyMTk2fQ.ePtZCfzIMNaeRCV1O5EtNMQ0myMBVffM9z95e4p9u24"

	smsfURL := fmt.Sprintf("/nsmsf-sms/v1/ue-contexts/%s/sendsms", supi)

	hdr := http.Header{}
	hdr.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "multipart/related;boundary=Boundary")

	reqBody := MakeUplinkMsg(cptype, msgRef, cc)
	uplinkResp, respData, err := client.Call("POST", smsfURL, hdr, reqBody)

	if err != nil {
		fmt.Printf("Send Fail Uplink Msg, SUPI:%s\n", supi)
		//	s.s_metrics.UplinkStop(now, uplinkResp.StatusCode) //err 일경우 stat 쌓으면, resp가 null 이기 때문에.. 죽는 현상 발생
		return
	}

	if verbose == true {
		fmt.Printf("resp code : %d, data : %s\n", uplinkResp.StatusCode, string(respData))
		fmt.Println("< END(UPLINK SEND) Service > -----------------------------------------------------------------------")
	}

	s.s_metrics.UplinkStop(now, uplinkResp.StatusCode)
	return
}

func MakeFailNotiMsg(cause string, notiUri string) []byte {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	if verbose == true {
		fmt.Println("< AMF(FAILNOTI SEND) > -----------------------------------------------------------------------")
	}

	request := msg5g.N1N2MsgTxfrFailureNotification{
		Cause:          cause,
		N1n2MsgDataUri: notiUri,
	}

	reqBody, err := json.Marshal(request)

	if err != nil {
		fmt.Printf("json.Mashal Err : %s\n", err)
		panic(err)
	}

	if verbose == true {
		fmt.Printf("cause : %s, notiUri : %s\n", cause, notiUri)
		fmt.Println("< END(FAILNOTI SEND) Service > -----------------------------------------------------------------------")
	}

	return reqBody
}

func MakeUplinkMsg(cptype int, msgRef byte, cc int) []byte {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var smsRecordId string
	var contentsId string

	if cptype == 0 {
		smsRecordId = "cpack-123"
		contentsId = "<cpack-123@amf.com>"
	} else {
		smsRecordId = "cpdata-123"
		contentsId = "<cpdata-123@amf.com>"
	}

	request := msg5g.UplinkSMS{
		SmsRecordID: smsRecordId,
		//SmsPayloads: []RefToBinaryData{
		SmsPayloads: []msg5g.RefToBinaryData{
			{ContentID: contentsId},
		},
		Gpsi:       "msisdn-01044445555",
		AccessType: "3GPP",
	}

	reqBody, err := json.Marshal(request)

	if err != nil {
		fmt.Printf("json.Mashal Err : %s\n", err)
		panic(err)
	}

	var b bytes.Buffer
	w := related.NewWriter(&b)
	w.SetBoundary("Boundary")

	rootPart, err := w.CreateRoot("", "application/json", nil)
	if err != nil {
		fmt.Printf("w.CreateRoot Err : %s\n", err)
	}
	rootPart.Write(reqBody)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/vnd.3gpp.sms")

	nextPart, err := w.CreatePart(contentsId, header)
	if err != nil {
		fmt.Printf("CreatePart Err : %s", err)
	}

	var cpdata []byte

	if cptype == 0 {
		/* Make CP-ACK */
		cpdata = make([]byte, 100, 200)
		cpdata = []byte{0x89, 0x04}
		if verbose == true {
			fmt.Println("===> Set CP-ACK")
		}
	} else {
		var RpData rpdu.RPDU

		RpData.Direction = rpdu.DIRECTION_MS_N
		RpData.MessageReference = msgRef //1byte

		if cc > 0 {
			if verbose == true {
				fmt.Println("===> Set RP-ERROR in CP-DATA ")
			}
			/* Make RP-ERROR */
			RpData.MessageType = rpdu.RP_ERROR_MS_N
			RpData.RpError = rpdu.RP_PROTOCOL_ERROR_UNSPECIFIED
		} else {
			if verbose == true {
				fmt.Println("===> Set RP-ACK in CP-DATA ")
			}
			/* Make RP-ACK */
			RpData.MessageType = rpdu.RP_ACK_MS_N
		}

		var cpData cpdu.CpEncode

		cpData.Direction = cpdu.DIRECTION_MS_N
		cpData.ProtocolDiscr = cpdu.CP_SMS_MESSAGES //24.007 -> sms
		cpData.TransactionId = cpdu.CP_TRANSACTION_IDENTIFIER_FLAG0

		if cc > 0 {
			cpData.LengthInd = 5
			cpData.CpData[0] = RpData.MessageType
			cpData.CpData[1] = RpData.MessageReference
			cpData.CpData[2] = 1
			cpData.CpData[3] = RpData.RpError
			cpData.CpData[4] = 0
		} else {
			cpData.LengthInd = 2
			cpData.CpData[0] = RpData.MessageType
			cpData.CpData[1] = RpData.MessageReference
		}

		cpdata = cpdu.EncodingData(cpData)
	}

	nextPart.Write(cpdata)

	if err := w.Close(); err != nil {
		fmt.Printf("w.Close() err: %s\n", err)
	}

	if verbose == true {
		fmt.Printf("Body : %s\n", b.String())
	}

	return b.Bytes()
}

func main() {

	path := os.Getenv("AMF_SIM_CONFIG_FILE")
	cfg := hocon.New(path)

	maxConnCnt := cfg.GetInt("server.SetMaxOpenConns", 100)

	runtime.GOMAXPROCS(maxConnCnt + runtime.NumCPU())

	ulog.Info("Start AMF Service")
	s := NewAmfServer(cfg)

	if s == nil {
		fmt.Println("make Amf-Server Fail")
		return
	}
	s.Start()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

}
