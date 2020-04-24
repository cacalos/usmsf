package httpif

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"strconv"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/dao"

	cdr "camel.uangel.com/ua5g/usmsf.git/implements/cdrmgr"
	"camel.uangel.com/ua5g/usmsf.git/implements/tcpmgr"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"

	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

type IfServer struct {
	common.HTTPServer

	redisDao    dao.RedisSubDao
	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server
	tcpInfo     *tcpmgr.TcpServer
	cdr         *cdr.CdrMgr
	svctype     string

	httpsAddr string
}

const TYPE_MO = "1"
const TYPE_MT = "2"

const CDR_SUCC = "1"
const CDR_FAIL = "2"

const IfServerSvcName = "HTTPIF"

func NewIfServerDia(cfg uconf.Config,
	tcpinfo *tcpmgr.TcpServer,
	traceMgr interfaces.TraceMgr,
	cdrMgr *cdr.CdrMgr,
) *IfServer {

	var cliConf common.HTTPCliConf

	httpConf := cfg.GetConfig("http-interface.http")
	smsfConf := cfg.GetConfig("httpif")

	cliConf.DialTimeout = smsfConf.GetDuration("http-interface.client-connection.timeout", time.Second*20)
	cliConf.DialKeepAlive = smsfConf.GetDuration("http-interface.client-connection.keep-alive", time.Second*20)
	cliConf.IdleConnTimeout = smsfConf.GetDuration("http-interface.client-connection.expire-time", 1*time.Minute)
	cliConf.InsecureSkipVerify = true

	s := &IfServer{
		tcpInfo: tcpinfo,
		cdr:     cdrMgr,
	}

	svctype := os.Getenv("MY_SERVICE_TYPE")
	if svctype == "" {
		return nil
	} else {
		s.svctype = svctype
	}

	s.Handler = echo.New()

	s.http2SvrCfg = &http2.Server{
		MaxHandlers:                  cfg.GetInt("http-interface.http.max-handler", 0),
		MaxConcurrentStreams:         uint32(cfg.GetInt("http-interface.http.max-concurrent-streams", 4000)),
		MaxReadFrameSize:             uint32(cfg.GetInt("http-interface.http.max-readframesize")),
		IdleTimeout:                  cfg.GetDuration("http-interface.http.idle-timeout", 100),
		MaxUploadBufferPerConnection: int32(cfg.GetInt("http-interface.http.maxuploadbuffer-per-connection", 65536)),
		MaxUploadBufferPerStream:     int32(cfg.GetInt("http-interface.http.maxuploadbuffer-per-stram", 65536)),
	}

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		httpPort := httpConf.GetInt("port", 8080)
		s.Addr = httpAddr + ":" + strconv.Itoa(httpPort)
		httpsvr := &http.Server{
			Addr:              s.Addr,
			Handler:           h2c.NewHandler(s.Handler, s.http2SvrCfg),
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
		}
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	}

	s.Handler.POST("/:s/:v/:n/:o", s.Handle)

	return s
}

func NewIfServer(cfg uconf.Config,
	tcpinfo *tcpmgr.TcpServer,
	redisdaoSet *dao.RedisDaoSet,
	traceMgr interfaces.TraceMgr,
) *IfServer {

	var cliConf common.HTTPCliConf

	httpConf := cfg.GetConfig("http-interface.http")
	smsfConf := cfg.GetConfig("httpif")

	cliConf.DialTimeout = smsfConf.GetDuration("http-interface.client-connection.timeout", time.Second*20)
	cliConf.DialKeepAlive = smsfConf.GetDuration("http-interface.client-connection.keep-alive", time.Second*20)
	cliConf.IdleConnTimeout = smsfConf.GetDuration("http-interface.client-connection.expire-time", 1*time.Minute)
	cliConf.InsecureSkipVerify = true

	s := &IfServer{
		redisDao: redisdaoSet.RedisSubDao,
		tcpInfo:  tcpinfo,
	}

	s.Handler = echo.New()

	s.http2SvrCfg = &http2.Server{
		MaxHandlers:                  cfg.GetInt("http-interface.http.max-handler", 0),
		MaxConcurrentStreams:         uint32(cfg.GetInt("http-interface.http.max-concurrent-streams", 4000)),
		MaxReadFrameSize:             uint32(cfg.GetInt("http-interface.http.max-readframesize")),
		IdleTimeout:                  cfg.GetDuration("http-interface.http.idle-timeout", 100),
		MaxUploadBufferPerConnection: int32(cfg.GetInt("http-interface.http.maxuploadbuffer-per-connection", 65536)),
		MaxUploadBufferPerStream:     int32(cfg.GetInt("http-interface.http.maxuploadbuffer-per-stram", 65536)),
	}

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		httpPort := httpConf.GetInt("port", 8080)
		s.Addr = httpAddr + ":" + strconv.Itoa(httpPort)
		httpsvr := &http.Server{
			Addr:              s.Addr,
			Handler:           h2c.NewHandler(s.Handler, s.http2SvrCfg),
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
		}
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	}

	s.Handler.POST("/:s/:v/:n/:o", s.Handle)

	return s
}

func (s *IfServer) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "HTTPIF start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the HTTPIF http://%v", s.Addr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve HTTPIF http://%v: error=%#v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

}

func (s *IfServer) Handle(ctx echo.Context) error {

	var err error

	service := ctx.Param("s")
	version := ctx.Param("v")
	operation := ctx.Param("o")
	supi := ctx.Param("n")

	if ctx.Request().Body != nil {
		defer ctx.Request().Body.Close()
	}

	switch service {
	case "map":
		if version != "v1" || len(supi) == 0 {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
		}

		switch ctx.Request().Method {
		case "POST":
			if operation == "rpresp" {
				s.HandleMtRPResp(ctx, supi)
			} else {
				loggers.ErrorLogger().Major("Unsupported Request Operation : %s", operation)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			}

		default:
			loggers.ErrorLogger().Major("Unsupported Request Method : %s", ctx.Request().Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
		}

	case "svc":
		if version != "v1" || len(supi) == 0 {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

		}

		switch ctx.Request().Method {
		case "POST":
			if operation == "rpdata" {
				s.HandleMoRPData(ctx, supi)
			} else {
				loggers.ErrorLogger().Major("Unsupported Request Operation : %s", operation)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			}

		default:
			loggers.ErrorLogger().Major("Unsupported Request Method : %s", ctx.Request().Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
		}
	default:
		loggers.ErrorLogger().Major("Unsupported Service : %s", service)
		err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

	}

	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
	}

	return err
}

func (s *IfServer) HandleMoRPData(c echo.Context, supi string) error {

	contentsType := c.Request().Header.Get("Content-Type")

	loggers.InfoLogger().Comment("Recv MO-RPDATA From SMSF-SVC-POD , supi:%s, contentsType:%s",
		supi, contentsType)

	if len(supi) == 0 || len(contentsType) == 0 {
		loggers.ErrorLogger().Major("HTTP msg missing parameter")

		//		s.cdr.WriteCdr(supi, timestamp, "1", "msisdn", CDR_FAIL) // 유입된 시간을 알수가 없네......

		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		c.Response().WriteHeader(http.StatusBadRequest)
		return json.NewEncoder(c.Response()).Encode(common.Header{Cause: "SMS_PAYLOAD_ERROR"})
	}

	smsContext := new(msg5g.MoSMS)

	if err := c.Bind(smsContext); err != nil {
		loggers.ErrorLogger().Major("GetRawData() Fail. No Body, err : %s", err)
		return s.RespondSystemError(c, errcode.BadRequest(c.Request().URL.Path))
	}
	// mo RPDATA를 받은 경우 알 수 있는 값.... + supi
	loggers.InfoLogger().Comment("rpdata[%s] : %x", supi, smsContext.Rpmsg) //발신 메시지
	loggers.InfoLogger().Comment("constanceID : %s", smsContext.ContentsId) //발신 ID
	loggers.InfoLogger().Comment("gpsi : %s", smsContext.Gpsi)              //착신 GPSI

	// Make and Send Msg to MAP pod
	loggers.InfoLogger().Comment("rpdata : %x", smsContext.Rpmsg)

	SendData, val := MakeMoSendData(smsContext, supi, tcpmgr.MO_MSG)

	//Error Code
	if val > 0 {

		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		c.Response().WriteHeader(http.StatusBadRequest)

		return json.NewEncoder(c.Response()).Encode(common.Header{Cause: "SMS_PAYLOAD_ERROR"})

	}

	err := s.SendToMsgProxy(SendData)

	if err != nil {

		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		c.Response().WriteHeader(http.StatusBadRequest)

		return json.NewEncoder(c.Response()).Encode(common.Header{Cause: "SMS_PAYLOAD_ERROR"})
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)

	return json.NewEncoder(c.Response()).Encode(nil)

}

func (s *IfServer) HandleMtRPResp(c echo.Context, supi string) error {
	var Result string

	now_in := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now_in.Year(), now_in.Month(), now_in.Day(),
		now_in.Hour(), now_in.Minute(), now_in.Second())

	notiCheck := false
	SendData := tcpmgr.MtData{}

	contentsType := c.Request().Header.Get("Content-Type")

	loggers.InfoLogger().Comment("Recv MT-RESPONSE From SMSF-SVC-POD, supi:%s, contentsType:%s", supi, contentsType)

	loggers.InfoLogger().Comment("name : %s ", supi)
	if len(supi) == 0 || len(contentsType) == 0 {
		loggers.ErrorLogger().Major("HTTP msg missing parameter")
		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		c.Response().WriteHeader(http.StatusBadRequest)

		return json.NewEncoder(c.Response()).Encode(common.Header{Cause: "SMS_PAYLOAD_ERROR"})

	}

	mtAck := new(msg5g.SmsResp)

	if err := c.Bind(mtAck); err != nil {
		loggers.ErrorLogger().Major("GetRawData() Fail. No Body, err : %s", err)
		return c.String(http.StatusBadRequest, "")
	}

	if contentsType == "application/json" {

		if mtAck.MsgType == "Failure-Notify" {

			notiCheck = true
		}

	} else {
		loggers.ErrorLogger().Major("Unknown Content-Type: %s", contentsType)
		return s.RespondBadRequest(c)
	}

	if s.svctype == "SIGTRAN" {
		// Get Redis Memory
		rval, redisData := s.redisDao.GetSubBySUPI(supi)
		if rval == -1 {
			loggers.ErrorLogger().Major("Dose not find Response Info in RedisDB. USER : %s", supi)
			return s.RespondBadRequest(c)
		}

		// Make and Send Msg to MAP pod
		loggers.InfoLogger().Comment("rpAck : %x", mtAck.Rpmsg)

		if notiCheck == true {
			SendData = s.MakeMtRespNotiSendData(mtAck, mtAck.Rpmsg, supi, tcpmgr.MT_RESP, redisData)

		} else {
			SendData = s.MakeMtRespSendData(mtAck, mtAck.Rpmsg, supi, tcpmgr.MT_RESP, redisData)
		}

		err := s.SendRespToMsgProxy(SendData)

		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			return c.JSON(http.StatusInternalServerError, nil)

		}

	} else if s.svctype == "DIAMETER" { // diameter
		// Make and Send Msg to MAP pod
		loggers.InfoLogger().Comment("rpAck : %x", mtAck.Rpmsg)

		if notiCheck == true {
			SendData = s.MakeMtRespNotiSendDataForDiameter(mtAck, mtAck.Rpmsg, supi, tcpmgr.MT_RESP)

		} else {
			SendData = s.MakeMtRespSendDataForDiameter(mtAck, mtAck.Rpmsg, supi, tcpmgr.MT_RESP)
		}

		err := s.SendRespToMsgProxyForDiameter(SendData)
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			return c.JSON(http.StatusInternalServerError, nil)

		}

	}

	//c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, nil)

	now_out := time.Now()
	doneTime := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now_out.Year(), now_out.Month(), now_out.Day(),
		now_out.Hour(), now_out.Minute(), now_out.Second())

	if mtAck.Result == 0 {
		Result = "1"
	} else {
		Result = "2"
	}

	OrigSupi := ""

	s.cdr.WriteCdr(TYPE_MT, OrigSupi, supi, timestamp, doneTime, Result)
	loggers.InfoLogger().Comment("Resp Message Send To SMSF_SVC_POD SUCC")
	return nil

}
