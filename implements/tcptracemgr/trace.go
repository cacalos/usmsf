package tcptracemgr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	uexec "camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

var loggers = common.SamsungLoggers()

type TraceServer struct {
	common.HTTPServer
	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server
	smsClient   *common.HTTPClient
	httpsAddr   string
}

const TraceSvcName = "Trace"

func NewTraceServer(cfg uconf.Config, traceMgr interfaces.TraceMgr) *TraceServer {

	var cliConf common.HTTPCliConf
	var smsClient *common.HTTPClient
	var err error

	httpConf := cfg.GetConfig("http-tracemgr.http")
	httpsConf := cfg.GetConfig("http-tracemgr.https")
	smsfConf := cfg.GetConfig("http-tracemgr")

	smschost := os.Getenv("EM_HOST")

	cliConf.DialTimeout = smsfConf.GetDuration("map-client.connection.timeout", time.Second*20)
	cliConf.DialKeepAlive = smsfConf.GetDuration("map-client.connection.keep-alive", time.Second*20)
	cliConf.IdleConnTimeout = smsfConf.GetDuration("map-client.connection.expire-time", 1*time.Minute)
	cliConf.InsecureSkipVerify = true

	if smschost != "" {
		smsClient, err = common.NewHTTPClient(&cliConf, "http", smschost, smschost, 1, traceMgr)
		if err != nil {
			loggers.ErrorLogger().Major("Failed to create SMS Client")
			return nil
		}
	}

	s := &TraceServer{
		smsClient: smsClient,
	}

	s.Handler = echo.New()

	s.http2SvrCfg = &http2.Server{}

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		httpPort := httpConf.GetInt("port", 8100)
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
		httpsPort := httpsConf.GetInt("port", 9110)
		tlscfg := cfg.GetConfig("http-tracemgr.tls")
		if tlscfg == nil {
			tlscfg = cfg.GetConfig("smsf.tls.internal-network")
			if tlscfg == nil {
				tlscfg = cfg.GetConfig("smsf.tls")
				if tlscfg == nil {
					loggers.ErrorLogger().Major("Not found TLS configuration (als-server.tls| sepp-tls.internal-ntework| sepp.tls)")
					return nil
				}
			}
		}

		certInfo, err := common.NewCertInfoByCfg(tlscfg)
		if err != nil {
			loggers.ErrorLogger().Major("Failed to create CertInfo: error=%#v", err.Error())
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

	s.CheckTrace()
	s.Handler.POST("/:s/:v/:o", s.Handle)

	return s
}

func (s *TraceServer) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		uexec.SafeGo(func() {
			waitchnl <- "Trace start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the Trace http://%v", s.Addr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve Trace http://%v: error=%#v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

	if s.httpsServer != nil {
		uexec.SafeGo(func() {
			waitchnl <- "Trace start https://" + s.httpsAddr
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the Trace https://%v", s.httpsAddr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve Trace https://%v: error=%#v", s.httpsAddr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

}

func (s *TraceServer) Handle(ctx echo.Context) error {

	var err error

	service := ctx.Param("s")
	version := ctx.Param("v")
	operation := ctx.Param("o")

	if ctx.Request().Body != nil {
		defer ctx.Request().Body.Close()
	}

	switch service {
	case "em":
		if version != "v1" || operation != "trace" {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

		}

		switch ctx.Request().Method {
		case "POST":
			if operation == "trace" {
				s.HandleTrace(ctx)
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

func (s *TraceServer) HandleTrace(c echo.Context) error {
	trace := [TRACE_REG_MAX]TraceInfo{}

	loggers.InfoLogger().Comment("Recv Trace Info From EM Trace")
	RawTraceBody, err := c.Request().GetBody()
	if err != nil {
		loggers.ErrorLogger().Major("Invalid Trace Payload Error")

		return s.ErrorBadRequest(c)
	}

	TraceBody, err := ioutil.ReadAll(RawTraceBody)

	if err != nil {
		loggers.ErrorLogger().Major("Invalid Trace Payload Error")

		return s.ErrorBadRequest(c)
	}

	loggers.InfoLogger().Comment("Trace Info : %s", string(TraceBody))

	err = json.Unmarshal(TraceBody, &trace)
	if err != nil {
		loggers.ErrorLogger().Major("JSON unmarshalling Error, err:%s", err.Error())
		c.JSON(http.StatusInternalServerError, nil)
		return err
	}

	uexec.SafeGo(func() {
		envpath := os.Getenv("HOME")

		tracePath := fmt.Sprintf("%s/home/bin/trace", envpath)

		for i := 0; i < TRACE_REG_MAX; i++ {

			if len(trace[i].Target) > 0 {

				if trace[i].Level != 1 {
					trace[i].Level = 1
				}

				level := strconv.Itoa(int(trace[i].Level))
				UntilTime := int64(time.Now().Unix()) - trace[i].Create_UnixTime + trace[i].Duration
				duration := strconv.Itoa(int(UntilTime))

				command := exec.Command(tracePath, trace[i].Target, duration, level)
				err = command.Run()
				if err != nil {
					loggers.ErrorLogger().Major("%s", err.Error())
				}

			}
		}
	})

	loggers.InfoLogger().Comment("Send Response 200 OK To EM Trace")
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlain)
	c.Response().WriteHeader(http.StatusOK)

	return json.NewEncoder(c.Response()).Encode(nil)

}

func (s *TraceServer) CheckTrace() {
	var EmSubAgentUrl string
	trace := [TRACE_REG_MAX]TraceInfo{}

	EmSubAgentUrl = fmt.Sprintf("/emsub/v1/trace")

	loggers.InfoLogger().Comment("Req Url : %s%s", s.smsClient.RootPath, EmSubAgentUrl)
	Resp, RespData, err := s.smsClient.Call("GET", EmSubAgentUrl, nil, nil)
	if err != nil {
		loggers.ErrorLogger().Major("Respnse err() : %s", err.Error())
		return
	}

	loggers.InfoLogger().Comment("Resp From EM : %d", Resp.StatusCode)

	if Resp.StatusCode > 300 {
		loggers.ErrorLogger().Major("EM Response Error : %d", Resp.StatusCode)
		return
	}

	err = json.Unmarshal(RespData, &trace)
	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		return
	}

	envpath := os.Getenv("HOME")
	tracePath := fmt.Sprintf("%s/home/bin/trace", envpath)

	for i := 0; i < TRACE_REG_MAX; i++ {
		if len(trace[i].Target) > 0 {

			if trace[i].Level != 1 {
				trace[i].Level = 1
			}

			level := strconv.Itoa(int(trace[i].Level))
			UntilTime := int64(time.Now().Unix()) - trace[i].Create_UnixTime + trace[i].Duration
			duration := strconv.Itoa(int(UntilTime))

			command := exec.Command(tracePath, trace[i].Target, duration, level)
			err = command.Run()
			if err != nil {
				loggers.ErrorLogger().Major("%s", err.Error())
			}
		}
	}

}
