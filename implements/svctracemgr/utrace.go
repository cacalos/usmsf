package svctracemgr

import (
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
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/utrace"

	"encoding/json"
	"strings"

	"io/ioutil"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

// TimeFormat Time Format
const TimeFormat = "2006-01-02 15:04:05"

// TimeMilliFormat Time Millisecond Format
const TimeMilliFormat = "2006-01-02 15:04:05.000"

const TRACE_REG_MAX = 10

type TraceInfo struct {
	Target   string `json:"target,omitempty"`
	Level    int    `json:"level,omitempty"`
	Duration int64  `json:"duration,omitempty"`
	//  Create_UnixTime time.Time `json:"create_unixtime"`
	Create_UnixTime int64 `json:"create_unixtime,omitempty"`
}

type TraceSvcPod struct {
	common.HTTPServer
	traceMgr    interfaces.TraceMgr
	httpServer  *http.Server
	httpsServer *http.Server
	smsClient   *common.HTTPClient
	httpsAddr   string
	//httpcli        uclient.HTTP
	//circuitBreaker uclient.HTTPCircuitBreaker
	http2SvrCfg   *http2.Server
	trace         [TRACE_REG_MAX]TraceInfo
	FileName      string
	traceRegFlag  [TRACE_REG_MAX]bool
	traceDuration [TRACE_REG_MAX]int64
	RegCount      int
	OnOff         bool
}

var loggers = common.SamsungLoggers()

const TraceSvcPodName = "TraceSvcPod"

func NewSvcPodTrace(cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
) *TraceSvcPod {

	var cliConf common.HTTPCliConf
	var smsClient *common.HTTPClient
	var err error

	httpConf := cfg.GetConfig("trace.http")
	httpsConf := cfg.GetConfig("svc-tracemgr.https")

	smschost := os.Getenv("EM_HOST")

	cliConf.DialTimeout = httpConf.GetDuration("connection.timeout", time.Second*20)
	cliConf.DialKeepAlive = httpConf.GetDuration("connection.keep-alive", time.Second*20)
	cliConf.IdleConnTimeout = httpConf.GetDuration("connection.expire-time", 1*time.Minute)
	cliConf.InsecureSkipVerify = true

	if smschost != "" {
		smsClient, err = common.NewHTTPClient(&cliConf, "http", smschost, smschost, 1, traceMgr)
		if err != nil {
			loggers.ErrorLogger().Major("Failed to create SMS Client: error=%#v", err.Error())
			return nil
		}
	}

	s := &TraceSvcPod{
		traceMgr:  traceMgr,
		smsClient: smsClient,
		OnOff:     cfg.GetConfig("trace.http").GetBoolean("onoff", false),
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
		tlscfg := cfg.GetConfig("svc-tracemgr.tls")
		if tlscfg == nil {
			tlscfg = cfg.GetConfig("svc.tls.internal-network")
			if tlscfg == nil {
				tlscfg = cfg.GetConfig("svc.tls")
				if tlscfg == nil {
					loggers.ErrorLogger().Major("Not found TLS configuration (als-server.tls| sepp-tls.internal-     ntework| sepp.tls)")
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

	//init Trace
	//	s.CheckTrace()

	s.Handler.POST("/:s/:v/:c", s.Handle)

	return s
}

func (s *TraceSvcPod) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "TraceSvcPod start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the TraceSvcPod http://%v", s.Addr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve TraceSvcPod http://%v: error=%#v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

	if s.httpsServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "TraceSvcPod start https://" + s.httpsAddr
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the TraceSvcPod https://%v", s.httpsAddr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve TraceSvcPod https://%v: error=%#v", s.httpsAddr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

}

/*
func (s *TraceSvcPod) CheckTrace() {
	var EmSubAgentUrl string
	//	EmSubAgentUrl = fmt.Sprintf("http://192.168.1.247:9292/em/v1/trace")

	EmSubAgentUrl = fmt.Sprintf("/emsub/v1/trace")

	loggers.InfoLogger().Comment("%s", EmSubAgentUrl)

	Resp, RespData, err := s.smsClient.Call("GET", EmSubAgentUrl, nil, nil)
	if err != nil {
		loggers.ErrorLogger().Major("Respnse err() : %s", err.Error())
		return
	}

	loggers.InfoLogger().Comment("Req Url : %s%s", s.smsClient.RootPath, EmSubAgentUrl)
	loggers.InfoLogger().Comment("Resp From EM : %d", Resp.StatusCode)

	err = json.Unmarshal(RespData, &s.trace)
	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		return
	}

	now := int64(time.Now().Unix())
	for i := 0; i < TRACE_REG_MAX; i++ {
		if len(s.trace[i].Target) > 0 {
			//	loggers.InfoLogger().Comment("Trace Target : %s", s.trace[i].Target)
			duration := s.trace[i].Create_UnixTime - now + s.trace[i].Duration
			TraceTime := now + duration
			s.traceRegFlag[i] = true
			s.traceDuration[i] = TraceTime
			s.RegCount++
		} else {
			s.traceRegFlag[i] = false
		}

	}

}
*/

func (s *TraceSvcPod) TraceValue(supi string) (*TraceInfo, string) {

	convSupi := strings.TrimLeft(supi, "imsi-")
	//	loggers.InfoLogger().Comment("Trace SUPI : %s", convSupi)
	for i := 0; i < TRACE_REG_MAX; i++ {
		//		loggers.InfoLogger().Comment("Trace Target(%d): %s", i, s.trace[i].Target)
		if s.trace[i].Target == convSupi {
			loggers.InfoLogger().Comment("Trace Target(%d): %s", i, s.trace[i].Target)
			return &s.trace[i], s.FileName
		}
	}
	return nil, ""
}

func (s *TraceSvcPod) Handle(ctx echo.Context) error {

	var err error

	service := ctx.Param("s")
	version := ctx.Param("v")
	operation := ctx.Param("c")

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
				err = s.HandleTrace(ctx)
				if err != nil {
					//			s.EnableTrace()
				}
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

func (s *TraceSvcPod) HandleTrace(ctx echo.Context) (err error) {

	traceRev := [TRACE_REG_MAX]TraceInfo{}

	loggers.InfoLogger().Comment("Trace Req service start")

	//	TraceBody, err := ctx.GetRawData()
	RawTraceBody, err := ctx.Request().GetBody()
	if err != nil {
		loggers.ErrorLogger().Major("HTTP body is error : %s", err)
		return s.RespondBadRequest(ctx)
	}

	TraceBody, err := ioutil.ReadAll(RawTraceBody)
	if err != nil {
		loggers.ErrorLogger().Major("HTTP body is error : %s", err)
		return s.RespondBadRequest(ctx)
	}

	err = json.Unmarshal(TraceBody, &traceRev)
	if err != nil {
		loggers.ErrorLogger().Major("JSON unmarshalling Error, err:%s", err)
		return s.RespondBadRequest(ctx)
	}

	s.trace = traceRev

	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlain)
	ctx.Response().WriteHeader(http.StatusOK)

	return json.NewEncoder(ctx.Response()).Encode(nil)

}

// EnableTrace implements utrace.TraceChecker.EnableTrace
// Trace Key를 설정한다.
// NRF에서는 중앙 configuration에서 가져오므로 사용하지 않는다.
func (s *TraceSvcPod) EnableTrace(keyType string, value string, expire time.Duration, level ulog.LogLevel) error {
	return errcode.New(http.StatusNotAcceptable, "Not acceptable call")
}

// DisableTrace implements utrace.TraceChecker.DisableTrace
// Trace Key를 삭제한다.
// NRF에서는 중앙 configuration에서 가져오므로 사용하지 않는다.
func (s *TraceSvcPod) DisableTrace(keyType string, value string) error {
	return errcode.New(http.StatusNotAcceptable, "Not acceptable call")
}

// CheckTrace implements utrace.TraceChecker.CheckTrace
// Trace 해야 하는지 확인한다.

func (s *TraceSvcPod) CheckTrace(keyType string, value string) (bool, ulog.LogLevel) {
	return s.Check(keyType, value, time.Now())
}

// Check 전달된 Key의 Trace Key 정보가 있는지 확인해 있으면 반환한다.
func (s *TraceSvcPod) Check(kind, key string, now time.Time) (bool, ulog.LogLevel) {
	return true, ulog.InfoLevel
	/*
		if m.AllTrace {
			return true, ulog.InfoLevel
		}
		trckeys := m.keys.Load().(map[string]TraceKeyMap)
		keymap, exists := trckeys[kind]
		if keymap == nil || !exists {
			return false, ulog.InfoLevel
		}
		trckey, exists := keymap[key]
		if exists && trckey != nil && now.Before(trckey.Expire) {
			return true, trckey.Level
		}
		if m.AllKey != "" {
			trckey, exists = keymap[m.AllKey]
			if exists && trckey != nil && now.Before(trckey.Expire) {
				return true, trckey.Level
			}
		}
		return false, ulog.InfoLevel
	*/
}

func MiddleWare(trace utrace.Trace, apiname string) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			// 인자로 넘어오는 req 는 trace context 가 추가된 request 이고 , Body 는 read 가능합니다.
			utrace.HttpHandlerPreservIO(trace, apiname, func(spanContext utrace.SpanContext, res http.ResponseWriter, req *http.Request) {

				spanContext.SetTraceKey("supi", ctx.Param("n"))

				// req 가 바뀌었기 때문에 , echo의 context의 request 도 변경해 주어야 합니다.
				ctx.SetRequest(req)

				// 그 후에 실제 handler를 호출해 주면 됩니다.

				if err := next(ctx); err != nil {
					ctx.Error(err)

				}
			})(ctx.Response().Writer, ctx.Request()) // <- utrace.HttpHandlerPreservIO 는 http.HandlerFunc를 리턴해 주기 때문에 , 호출을 해주어야 실행이 됩니다.

			return nil

		}

	}

}

/*
func GinHTTPTraceHandler(trace utrace.Trace, serviceName string) *gin.HandlerFunc {
	return func(ctx *gin.Context) {

		utrace.HttpHandlerPreservIO(trace, serviceName, func(spanContext utrace.SpanContext, res http.ResponseWriter, req *http.Request) {

			//	ervice := ctx.Param("s")
			//	version := ctx.Param("v")
			//	operation := ctx.Param("o")
			supi := ctx.Param("n")
			//			spanContext.SetTags(ulog.InfoLevel, utypes.Map{"hello": "world"})

			ulog.Info("supi supi = %v", supi)
			spanContext.SetTraceKey("supi", supi)

			//ulog.Info("trace = %v", trace)
			//			ulog.Info("span context = %v, is enabled = %t", spanContext, spanContext.TracingState())
			//			ulog.Info("QueryGEt== %s", ctx.Request.URL.String())

			ulog.Info("utrace.HeaderStringer(ctx.Request.Header) :\n %v ", utrace.HeaderStringer(ctx.Request.Header))
			//spanContext.LogFields(ulog.InfoLevel, utypes.Map{
			//		"header": utrace.HeaderStringer(ctx.Request.Header),
			//})

			ctx.Request = req
			ctx.Next()
		})(ctx.Writer, ctx.Request)
	}
}
*/
