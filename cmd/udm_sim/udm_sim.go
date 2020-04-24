package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/hocon"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

//var logger = ulog.GetLogger("com.uangel.usmsf.udmsim")

// RespondSystemError System Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondSystemError(ctx *gin.Context, err error) error {
	pd := msg5g.SystemError(ctx.Request.URL.String(), err)
	return s.RespondProblemDetails(ctx, pd)
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

//HTTPServer HTTP 기반 서비스 서버의 공통 객체
type HTTPServer struct {
	Addr               string
	Handler            *gin.Engine
	authRequired       bool     //Authenticiation이 요구 되는지 여부
	authCredentials    [][]byte // slice with base64-encoded credentials
	probeResistDomain  string
	probeResistEnabled bool
}

type SDMResp struct {
	SupportFeatures     string `json:"supportFeatures,omitempty"`
	MtSmsSubscribed     bool   `json:"mtSmsSubScribed"`
	MtSmsBarringAll     bool   `json:"mtSmsBarringAll"`
	MtSmsBarringRoaming bool   `json:"mtSmsBarringRoaming"`
	MoSmsSubscribed     bool   `json:"moSmsSubScribed"`
	MoSmsBarringAll     bool   `json:"moSmsBarringAll"`
	MoSmsBarringRoaming bool   `json:"moSmsBarringRoaming"`
	SharedSmsMngDataIds string `json:"sharedSmsMngDataIds,omitempty"`
}

type UdmServer struct {
	HTTPServer
	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server

	metrics *HTTPMetrics

	httpsAddr string

	SdmResp SDMResp
}

const IfServerSvcName = "UDMSIM"

var subsResp bool
var sdmgetResp bool
var verbose bool

func NewUdmServer(cfg uconf.Config) *UdmServer {

	httpConf := cfg.GetConfig("udm-sim.http")
	httpsConf := cfg.GetConfig("udm-sim.https")
	udmConf := cfg.GetConfig("udmsim")

	s := &UdmServer{}

	if udmConf != nil {
		s.SdmResp.MtSmsSubscribed = udmConf.GetBoolean("MtSmsSubscribed", true)
		s.SdmResp.MtSmsBarringAll = udmConf.GetBoolean("MtSmsBarringAll", false)
		s.SdmResp.MtSmsBarringRoaming = udmConf.GetBoolean("MtSmsBarringRoaming", false)
		s.SdmResp.MoSmsSubscribed = udmConf.GetBoolean("MoSmsSubscribed", true)
		s.SdmResp.MoSmsBarringAll = udmConf.GetBoolean("MoSmsBarringAll", false)
		s.SdmResp.MoSmsBarringRoaming = udmConf.GetBoolean("MoSmsBarringRoaming", false)
	}

	subsResp = udmConf.GetBoolean("subsresponse", false)
	sdmgetResp = udmConf.GetBoolean("udmsim.sdmgetresponse", false)

	verbose = udmConf.GetBoolean("verbose", false)

	//	AccessType = udmConf.GetBoolean("accesstype", false)
	//	DeactResp = udmConf.GetBoolean("deactresponse", false)

	s.Handler = gin.New()

	ishttp2 := httpConf.GetBoolean("ishttp2", true)

	if ishttp2 == false {
		s.http2SvrCfg = &http2.Server{
			MaxHandlers:          cfg.GetInt("udm-sim.https.max-handler", 0),
			MaxConcurrentStreams: uint32(cfg.GetInt("udm-sim.https.max-concurrent-streams", 20000)),
		}
	} else {
		s.http2SvrCfg = &http2.Server{}

	}

	s.metrics = NewHTTPMetrics()

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		httpPort := httpConf.GetInt("port", 8081)
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
		httpsPort := httpsConf.GetInt("port", 9091)
		tlscfg := cfg.GetConfig("udm-sim.tls")
		if tlscfg == nil {
			tlscfg = cfg.GetConfig("udm.tls.internal-network")
			if tlscfg == nil {
				tlscfg = cfg.GetConfig("udm.tls")
				if tlscfg == nil {
					fmt.Println("Not found TLS configuration (als-server.tls| sepp-tls.internal-ntework| sepp.tls)")
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

	s.Handler.PUT("/:s/:v/:n/:o/:a", s.Handle)
	s.Handler.GET("/:s/:v/:n/:o", s.Handle)
	s.Handler.DELETE("/:s/:v/:n/:o/:a", s.Handle)
	s.Handler.POST("/:s/:v/:n/:o", s.Handle)

	return s
}

func (s *UdmServer) ReportStat() {

	reportPeriod := 1
	start := time.Now()
	reportSec := start.Unix()

	for {
		now := time.Now()
		nowSecond := now.Unix()

		if int(nowSecond-reportSec) >= reportPeriod {
			reportSec = nowSecond
			fmt.Printf("[%v]\n", time.Now())
			go s.metrics.Report(true, true, false)
			fmt.Printf("\n")
		}

		time.Sleep(1 * time.Second)
	}
}

func (s *UdmServer) Start() {
	if s.httpServer != nil {
		go func() {
			err := s.httpServer.ListenAndServe()
			if err != nil {
				fmt.Printf("Shutting down the UDMSIM http://%v\n", s.Addr)
				//s.controller.Close()
			} else {
				fmt.Printf("Failed to listen and serve UDMSIM http://%v\n", s.Addr)
			}
		}()
	}

	if s.httpsServer != nil {
		go func() {
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				fmt.Printf("Shutting down the UDMSIM https://%v\n", s.httpsAddr)
			} else {
				fmt.Printf("Failed to listen and serve UDMSIM https://%v\n", s.httpsAddr)
			}
		}()
	}
}

func (s *UdmServer) Handle(ctx *gin.Context) {

	var err error

	service := ctx.Param("s")
	version := ctx.Param("v")
	operation := ctx.Param("o")
	supi := ctx.Param("n")
	accessType := ctx.Param("a")

	if ctx.Request.Body != nil {
		defer ctx.Request.Body.Close()
	}

	switch service {
	case "ue-Contexts":
		if len(supi) == 0 {
			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

		}

		if ctx.Request.Method == "PUT" {
			s.HandleueContexts(ctx, supi)
		} else {
			fmt.Printf("Unsupported Request Method : %s\n", ctx.Request.Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

		}

	case "nudm-uecm":
		if version != "v1" || len(supi) == 0 {
			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
		}

		switch ctx.Request.Method {
		case "PUT":
			if operation == "registrations" {
				if accessType == "smsf-3gpp-access" {
					s.handleRegistrations(ctx, supi)
				} else if accessType == "smsf-non-3gpp-access" {
					s.HandleNonRegistraions(ctx, supi)
				} else {
					fmt.Printf("Unsupported Request Operation : %s\n", operation)
					err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
				}
			}
		case "DELETE":
			if operation == "registrations" {
				if accessType == "smsf-3gpp-access" {
					s.handleDeleteRegistrations(ctx, supi)
				} else if accessType == "smsf-non-3gpp-access" {
					s.handleDeleteNonRegistraions(ctx, supi)
				} else {
					fmt.Printf("Unsupported Request Operation : %s\n", operation)
					err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
				}
			}

		default:
			fmt.Printf("Unsupported Request Method : %s\n", ctx.Request.Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
		}

	case "nudm-sdm":
		if version != "v1" || len(supi) == 0 {
			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))
		}

		switch ctx.Request.Method {
		case "POST":
			if operation == "sdm-subscriptions" {
				s.handleSdmSubscription(ctx, supi)
			}
		case "GET":
			if operation == "sms-mng-data" {
				s.handleSdmGet(ctx, supi)
			}
		case "DELETE":
			if operation == "sdm-subscriptions" {
				s.handleSdmUnSubscription(ctx, supi)
			}
		default:

			fmt.Printf("Unsupported Request ver : %s or operation : %s\n", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

		}

	default:
		fmt.Printf("Unsupported Service : %s\n", service)
		err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request.URL.Path))

	}

	if err != nil {
		fmt.Printf(err.Error())
	}

}

func (s *UdmServer) HandleueContexts(c *gin.Context, supi string) {

	fmt.Printf("UserInfo : %s\n", supi)

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("err : %s\n", err.Error())
	}

	fmt.Printf("Message : %s\n", message)

	c.String(http.StatusOK, "PUT OK")

}

func (s *UdmServer) handleRegistrations(c *gin.Context, supi string) {
	now := s.metrics.Start()

	var code int

	if verbose == true {
		fmt.Println("< UDM(UECM ADD) Service > -----------------------------------------------------------------------")
		fmt.Printf("PUT UECM Request : User - %s\n", supi)
	}

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("Error : %s\n", err.Error())
	}

	if verbose == true {
		fmt.Printf("Message : %s\n", message)
	}

	//c.JSON(http.StatusCreated, message)

	if subsResp != true {
		c.String(http.StatusOK, "PUT OK")
		code = http.StatusOK
	} else {
		c.String(http.StatusBadRequest, "PUT TEST FAIL")
		code = http.StatusBadRequest
	}

	if verbose == true {
		fmt.Println("< END(UECM ADD) Service > -----------------------------------------------------------------------")
	}

	s.metrics.RegistrationStop(now, code)
}

func (s *UdmServer) HandleNonRegistraions(c *gin.Context, supi string) {
	now := s.metrics.Start()

	var code int

	if verbose == true {
		fmt.Println("< UDM(UECM ADD) Service > -----------------------------------------------------------------------")
		fmt.Printf("PUT UECM Request : User - %s\n", supi)
	}

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("Error : %s\n", err.Error())
	}

	//c.JSON(http.StatusCreated, message)
	if subsResp != true {
		c.String(http.StatusOK, "PUT OK")
		code = http.StatusOK
	} else {
		c.String(http.StatusBadRequest, "PUT TEST FAIL")
		code = http.StatusBadRequest
	}

	if verbose == true {
		fmt.Printf("Message : %s\n", message)
		fmt.Print("< END(UECM ADD) Service > -----------------------------------------------------------------------")
	}

	s.metrics.RegistrationStop(now, code)
}

func (s *UdmServer) handleDeleteRegistrations(c *gin.Context, supi string) {
	now := s.metrics.Start()

	var code int

	if verbose == true {
		fmt.Println("< UDM(UECM DEL) Service > -----------------------------------------------------------------------")
		fmt.Printf("DELETE UECM Request : User - %s\n", supi)
	}

	if subsResp != true {
		c.String(http.StatusOK, "DELETE OK")
		code = http.StatusOK
	} else {
		c.String(http.StatusBadRequest, "Delete TEST FAIL")
		code = http.StatusBadRequest
	}

	if verbose == true {
		fmt.Println("< END(UECM DEL) Service > --------------------------------------------------------------------")
	}
	s.metrics.DeRegistrationStop(now, code)
}

func (s *UdmServer) handleDeleteNonRegistraions(c *gin.Context, supi string) {
	now := s.metrics.Start()
	var code int

	if verbose == true {
		fmt.Println("< UDM(UECM DEL) Service > -----------------------------------------------------------------------")
		fmt.Printf("DELETE UECM Request : User - %s\n", supi)
	}

	if subsResp != true {
		c.String(http.StatusOK, "DELETE OK")
		code = http.StatusOK
	} else {
		c.String(http.StatusBadRequest, "Delete TEST FAIL")
		code = http.StatusBadRequest
	}

	if verbose == true {
		fmt.Println("< END(UECM DEL) Service > --------------------------------------------------------------------")
	}
	s.metrics.DeRegistrationStop(now, code)
}

func (s *UdmServer) handleSdmGet(c *gin.Context, supi string) {
	now := s.metrics.Start()

	var code int
	if verbose == true {
		fmt.Println("< UDM(SDM GET) Service > -----------------------------------------------------------------------")
		fmt.Printf("SDM GET Request : User - %s\n", supi)
	}

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("err : %s\n", err.Error())
	}

	if verbose == true {
		fmt.Printf("message : %s\n", message)
		fmt.Printf("s.SdmResp.MtSmsSubscribed : %+v\n", s.SdmResp.MtSmsSubscribed)
	}

	sdmResp := SDMResp{
		MtSmsSubscribed:     s.SdmResp.MtSmsSubscribed,
		MtSmsBarringAll:     s.SdmResp.MtSmsBarringAll,
		MtSmsBarringRoaming: s.SdmResp.MtSmsBarringRoaming,
		MoSmsSubscribed:     s.SdmResp.MoSmsSubscribed,
		MoSmsBarringAll:     s.SdmResp.MoSmsBarringAll,
		MoSmsBarringRoaming: s.SdmResp.MoSmsBarringRoaming,
	}

	if sdmgetResp != true {
		c.JSON(http.StatusOK, sdmResp)
		code = http.StatusOK
	} else {
		c.JSON(http.StatusNotFound, "")
		code = http.StatusNotFound
	}

	if verbose == true {
		fmt.Println("< END(SDM GET) Service > --------------------------------------------------------------------")
	}

	s.metrics.SdmGetStop(now, code)
}

func (s *UdmServer) handleSdmSubscription(c *gin.Context, supi string) {
	now := s.metrics.Start()

	if verbose == true {
		fmt.Println("< UDM(SUBSCRIBE) Service > -----------------------------------------------------------------------")
		fmt.Printf("POST SDM Subscription : User - %s\n", supi)
	}

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("err : %s\n", err.Error())
	}

	c.String(http.StatusOK, "POST sdm_subscriptions OK")
	if verbose == true {
		fmt.Printf("message : %s\n", message)
		fmt.Println("< END(SUBSCRIBE) Service > --------------------------------------------------------------------")
	}

	s.metrics.SubscriptionStop(now, http.StatusOK)
}

func (s *UdmServer) handleSdmUnSubscription(c *gin.Context, supi string) {

	now := s.metrics.Start()

	if verbose == true {
		fmt.Println("< UDM(UNSUBSCRIBE) Service > -----------------------------------------------------------------------")
		fmt.Printf("DELETE SDM Subscription : User - %s\n", supi)
	}

	message, err := c.GetRawData()
	if err != nil {
		fmt.Printf("err : %s\n", err.Error())
	}

	c.String(http.StatusOK, "DELETE sdm_subscriptions OK")
	if verbose == true {
		fmt.Printf("message : %s\n", message)
		fmt.Println("< END(UNSUBSCRIBE) Service > --------------------------------------------------------------------")
	}

	s.metrics.UnSubscriptionStop(now, http.StatusOK)
}

func main() {

	path := os.Getenv("UDM_SIM_CONFIG_FILE")
	cfg := hocon.New(path)

	maxConnCnt := cfg.GetInt("server.SetMaxOpenConns", 100)

	runtime.GOMAXPROCS(maxConnCnt + runtime.NumCPU())

	fmt.Println("Start UDM Server Service")
	s := NewUdmServer(cfg)
	s.Start()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
