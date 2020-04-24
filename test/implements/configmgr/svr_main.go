package configmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

type ConfigServer struct {
	common.HTTPServer
	httpServer  *http.Server
	http2SvrCfg *http2.Server

	router uclient.HTTPRouter
	cli    *common.HTTPClient

	cliConf common.HTTPCliConf
	useFlag string
	scheme  string

	httpsAddr  string
	ConfigPort string
	uccmshost  string
}

func (s *ConfigServer) LoadConfig(cfg uconf.Config) (err error) {

	smsfConf := cfg.GetConfig("http-configmgr")

	if smsfConf != nil {

		s.cliConf.DialTimeout = smsfConf.GetDuration("map-client.connection.timeout", time.Second*20)
		s.cliConf.DialKeepAlive = smsfConf.GetDuration("map-client.connection.keep-alive", time.Second*20)
		s.cliConf.IdleConnTimeout = smsfConf.GetDuration("map-client.connection.expire-time", 2*time.Minute)

		s.cliConf.InsecureSkipVerify = true

		s.useFlag = smsfConf.GetString("use", "map")
		s.scheme = smsfConf.GetString("scheme", "http")

	} else {
		return errors.New("Init Fail : HTTP CONFIG(Svc)")

	}
	return nil
}

func (s *ConfigServer) SetHTTPConfig(cfg uconf.Config) (err error) {
	httpConf := cfg.GetConfig("http-configmgr.http")
	httpPort := httpConf.GetInt("port", 8090)
	s.ConfigPort = strconv.Itoa(httpPort)

	s.http2SvrCfg = &http2.Server{}

	if httpConf != nil {
		httpAddr := httpConf.GetString("address", "")
		s.Addr = httpAddr + ":" + s.ConfigPort
		httpsvr := &http.Server{
			Addr:    s.Addr,
			Handler: h2c.NewHandler(s.Handler, s.http2SvrCfg),
		}
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	} else {
		return errors.New("Set HTTP Config Fail")
	}

	return nil
}

func NewConfigServer(
	cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
	httpcli uclient.HTTP,
	cli *ConfigClient,
) *ConfigServer {

	var err error
	s := &ConfigServer{
		//	router: cli.router,
		//	cli: cli.cli,
	}
	s.Handler = echo.New()

	err = s.LoadConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL LOAD -> CONFIG(ConfigMgr)")
		return nil
	}

	err = s.uccmshostConfig(httpcli)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL LOAD -> uccmshostConfig(ConfigMgr)")
		return nil
	}

	err = s.SetHTTPConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL LOAD -> HTTP_CONFIG(ConfigMgr)")
		return nil
	}

	s.Handler.POST("/:s/:o", s.Handle)

	return s
}

func (s *ConfigServer) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "CONFIGMGR start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the CONFIGMGR http://%v", s.Addr)
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve CONFIGMGR http://%v: error=%#v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

}

func (s *ConfigServer) Handle(ctx echo.Context) error {

	var err error

	service := ctx.Param("s")
	operation := ctx.Param("o")

	if ctx.Request().Body != nil {
		defer ctx.Request().Body.Close()
	}

	switch service {
	case "test":
		switch ctx.Request().Method {
		case "POST":
			if operation == "watch" {
				err = s.HandleMethodNotifyPOST(ctx)
				if err != nil {
					loggers.ErrorLogger().Major("%s", err.Error())
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

func (s *ConfigServer) uccmshostConfig(httpcli uclient.HTTP) (err error) {

	s.uccmshost = os.Getenv("CONF_POD_HOST")

	loggers.InfoLogger().Comment("Get Env : %s", s.uccmshost)

	if s.uccmshost != "" {
		if s.scheme != "" {
			if s.scheme != "http" {
				s.router = uclient.HTTPRouter{
					//	Scheme:           "http",
					Servers: []string{s.uccmshost},
					Client:  httpcli,
				}
			} else {
				s.cli, err = common.NewHTTPClient(&s.cliConf, "http", s.uccmshost, s.uccmshost, 1, nil)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return errors.New("Set FAil : uccmshost Config")
	}

	return nil
}

func (s *ConfigServer) CloseGracefully() error {
	s.UnsubscribeToUCCMS()
	return nil
}

func (s *ConfigServer) UnsubscribeToUCCMS() {

	_, watchMap := GetWatchIdMap()

	for _, watchId := range watchMap {
		loggers.InfoLogger().Comment("watchId: %s", watchId)

		DeleteURL := fmt.Sprintf("/app/v1/watch/%s", watchId)

		hdr := http.Header{}
		hdr.Add("accept", "application/json")
		hdr.Add("Content-Type", "application/json")

		watchDel := WatchDelete{
			WatchId: watchId,
		}

		loggers.InfoLogger().Comment("Delete WatchId : %s", watchId)
		reqBody, err := json.Marshal(watchDel)
		if err != nil {
			return
		}

		if s.scheme != "http" {
			DelResp, err := s.router.SendRequest(context.Background(), DeleteURL, http.MethodDelete, hdr, reqBody, 2*time.Second)
			if err != nil {
				loggers.ErrorLogger().Major("WATCH PostResp Response err() : %s ", err.Error())
				return
			}
			loggers.InfoLogger().Comment("Result Code : %d", DelResp.StatusCode)
			//err = json.Unmarshal(bodyOfJSONPart, &uplinkMsg)
		} else {

			loggers.InfoLogger().Comment("Send To Delete watch URL : %s:%s%s", http.MethodDelete, s.cli.RootPath, DeleteURL)
			DelResp, _, err := s.cli.Call(http.MethodDelete, DeleteURL, hdr, reqBody)
			if err != nil {
				loggers.ErrorLogger().Major("WATCH PostResp Response err() : %s ", err.Error())
				return
			}

			loggers.InfoLogger().Comment("Delete Watch Result Code : %d", DelResp.StatusCode)

		}

	}

}
