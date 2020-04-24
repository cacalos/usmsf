package controller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"errors"

	"github.com/labstack/echo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/xerrors"

	jwt "github.com/dgrijalva/jwt-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/philippfranke/multipart-related/related"

	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/uregi"
	"camel.uangel.com/ua5g/ulib.git/utrace"
	"camel.uangel.com/ua5g/ulib.git/utypes"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/dao"
	cpdu "camel.uangel.com/ua5g/usmsf.git/endecoder/cpdu"
	rpdu "camel.uangel.com/ua5g/usmsf.git/endecoder/rpdu"
	"camel.uangel.com/ua5g/usmsf.git/implements/configmgr"
	"camel.uangel.com/ua5g/usmsf.git/implements/db"
	"camel.uangel.com/ua5g/usmsf.git/implements/svctracemgr"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

var loggers = common.SamsungLoggers()

var callLogger = ulog.GetLogger("call.log.service")

// NFServer NFServer 구조체 정의
type NFServer struct {
	common.HTTPServer
	stats       *Stats
	redisDao    dao.RedisSubDao
	mysqlDao    dao.MySqlSubDao
	udmCli      scpcli.ScpClientDyn
	traceMgr    interfaces.TraceMgr
	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server
	scpClient   scpcli.ScpClientFactory
	traceInfo   *svctracemgr.TraceSvcPod
	trace       utrace.Trace
	routerdia   uclient.HTTPRouter
	routermap   uclient.HTTPRouter
	requires    NFServeriRequires

	cfg            uconf.Config
	httpcli        uclient.HTTP
	http2c         uclient.H2C
	circuitBreaker uclient.HTTPCircuitBreaker
	resolver       uregi.Resolver
	dbmgr          *db.DBManager

	connCnt            uint32
	httpsAddr          string
	fqdn               string
	nfId               string
	mnc                string
	mcc                string
	isdn               string
	name               string
	realm              string
	accessTokenChecker AccessTokenChecker
	notifyUrlAddr      string
}

// AccessTokenChecker : Access token 검사용 데이터 구조체
type AccessTokenChecker struct {
	shouldCheck    bool   // true인 경우에만 HTTP 헤더의 access token을 검사한다.
	signingMethod  string // "HMAC", "RSA", "RSAPSS", "ECDSA"
	keyFilePath    string
	key            []byte
	expectedClaims map[string]string
}

type NFServeriRequires struct {
	httpConf  uconf.Config
	httpsConf uconf.Config
	cliConf   common.HTTPCliConf
}

// "usmsf.conf -> smsf -> access-token-checker -> key-file"에 해당하는 파일로부터 key를 읽는다.
func (tokenChecker *AccessTokenChecker) getVerificationKeyFromFile(conf uconf.Config) error {

	tokenChecker.keyFilePath = conf.GetString("key-file", "")
	if tokenChecker.keyFilePath == "" {
		err := fmt.Errorf("\"key-file\" is missing")
		loggers.ConfigLogger().Major(err.Error())
		return err
	}

	tokenChecker.key = tokenChecker.getKeyFromFile()
	if tokenChecker.key == nil {
		err := fmt.Errorf("Failed to get key from file")
		loggers.ConfigLogger().Major(err.Error())
		return err
	}

	return nil
}

// "usmsf.conf -> smsf -> access-token-checker -> expected-claims" 설정 정보를 읽는다.
func (tokenChecker *AccessTokenChecker) getExpectedClaims(parentConf uconf.Config) error {

	conf := parentConf.GetConfig("expected-claims")
	if conf == nil {
		err := fmt.Errorf("Failed to get \"expected-claims\" from %#v", tokenChecker.keyFilePath)
		loggers.ConfigLogger().Major(err.Error())
		return err
	}

	if tokenChecker.expectedClaims == nil {
		tokenChecker.expectedClaims = make(map[string]string)
	}

	tokenChecker.expectedClaims["scope"] = conf.GetString("scope")

	return nil
}

// Key 파일로부터 byte 배열 타입의 key를 읽어 반환한다.
func (tokenChecker *AccessTokenChecker) getKeyFromFile() []byte {

	keyFile, err := os.Open(tokenChecker.keyFilePath)
	if err != nil {
		loggers.ConfigLogger().Major(err.Error())
		return nil
	}

	defer keyFile.Close()

	fileInfo, _ := keyFile.Stat()
	var size int64 = fileInfo.Size()
	pembytes := make([]byte, size)
	buffer := bufio.NewReader(keyFile)
	_, err = buffer.Read(pembytes)
	block, _ := pem.Decode(pembytes)

	return block.Bytes
}

func (s *NFServer) SvcHTTPConfig(cfg uconf.Config) (ret error) {
	s.requires.httpConf = cfg.GetConfig("nf-server.http")
	s.requires.httpsConf = cfg.GetConfig("nf-server.https")

	if s.requires.httpConf != nil {
		httpAddr := s.requires.httpConf.GetString("address", "")
		httpPort := s.requires.httpConf.GetInt("port", 8080)
		s.Addr = httpAddr + ":" + strconv.Itoa(httpPort)
		httpsvr := &http.Server{
			Addr:    s.Addr,
			Handler: h2c.NewHandler(s.Handler, s.http2SvrCfg),
		}
		http2.VerboseLogs = cfg.GetBoolean("nf-server.https.verbose-logs", false)
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	} else {
		ret = errors.New("Init Fail : HTTP CONFIG(Svc)")
		return ret
	}

	if s.requires.httpsConf != nil {
		httpsAddr := s.requires.httpsConf.GetString("address", "")
		httpsPort := s.requires.httpsConf.GetInt("port", 8444)
		tlscfg := cfg.GetConfig("nf-server.tls")
		if tlscfg == nil {
			tlscfg = cfg.GetConfig("smsf.tls.internal-network")
			if tlscfg == nil {
				tlscfg = cfg.GetConfig("smsf.tls")
				if tlscfg == nil {
					loggers.ConfigLogger().Major("Not found TLS configuration (als-server.tls| sepp-tls.internal-ntework| sepp.tls)")
					ret = errors.New("Not found TLS configuration")
					return ret
				}
			}
		}

		certInfo, err := common.NewCertInfoByCfg(tlscfg)
		if err != nil {
			loggers.ConfigLogger().Major("Failed to create CertInfo: %v", err.Error())
			ret = err
			return ret
		}

		s.httpsAddr = httpsAddr + ":" + strconv.Itoa(httpsPort)
		httpssvr := &http.Server{
			Addr:              s.httpsAddr,
			Handler:           s.Handler,
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      5 * time.Second,
			TLSConfig:         certInfo.GetServerTLSConfig(),
		}
		http2.VerboseLogs = cfg.GetBoolean("nf-server.https.verbose-logs", false)
		http2.ConfigureServer(httpssvr, s.http2SvrCfg)
		s.httpsServer = httpssvr
	} else {
		ret = errors.New("Init Fail : HTTPS CONFIG(Svc)")
		return ret
	}

	return nil
}

func (s *NFServer) SvcReadConfig(cfg uconf.Config) (ret error) {

	smsfConf := cfg.GetConfig("smsf")

	if smsfConf != nil {
		s.requires.cliConf.DialTimeout = smsfConf.GetDuration("sms-client.connection.timeout", time.Second*20)
		s.requires.cliConf.DialKeepAlive = smsfConf.GetDuration("sms-client.connection.keep-alive", time.Second*20)
		s.requires.cliConf.IdleConnTimeout = smsfConf.GetDuration("sms-client.connection.expire-time", 1*time.Minute)
		s.requires.cliConf.InsecureSkipVerify = true

		s.requires.cliConf.MaxHeaderListSize = uint32(smsfConf.GetInt("sms-client.MaxHeaderListSize", 1024000000))

		s.requires.cliConf.StrictMaxConcurrentStreams = smsfConf.GetBoolean("StrictMaxConcurrentStreams", false)

		s.connCnt = uint32(smsfConf.GetInt("sms-client.session-count", 5))

		s.fqdn = smsfConf.GetString("my-fqdn", "usmsf-svc.smsf.svc.cluster.local")
		s.nfId = smsfConf.GetString("my-nf-id", "abcde-1234-5678-abcd-ef120-1111")
		s.mnc = smsfConf.GetString("mnc", "450")
		s.mcc = smsfConf.GetString("mcc", "06")
		s.isdn = smsfConf.GetString("my-map-id", "821045001234")
		s.name = smsfConf.GetString("my-diameter-name", "smsf")
		s.realm = smsfConf.GetString("my-diameter-realm", "usmsf-svc.smsf.svc.cluster.local")

		err := s.getAccessTokenCheckerConfig(smsfConf)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Init Fail : Config Load(SvcReadConfig)")
	}

	return ret
}

// Access Token 검사에 필요한 config 값들을 읽는다.
func (s *NFServer) getAccessTokenCheckerConfig(cfg uconf.Config) error {

	var err error

	conf := cfg.GetConfig("access-token-checker")
	if conf == nil {
		loggers.ConfigLogger().Minor("\"access-token-checker\" not found: Not checking Access Token by default")
		s.accessTokenChecker.shouldCheck = false
		return nil
	}

	s.accessTokenChecker.shouldCheck = conf.GetBoolean("token-check", false)
	if !s.accessTokenChecker.shouldCheck {
		loggers.ConfigLogger().Comment("Set to NOT check Access Token")
		return nil
	}

	s.accessTokenChecker.signingMethod = conf.GetString("signing-method", "")
	if s.accessTokenChecker.signingMethod == "" {
		return errors.New("signing-method is missing")
	}

	err = s.accessTokenChecker.getVerificationKeyFromFile(conf)
	if err != nil {
		return errors.New("Failed to get verification key from config file")
	}

	err = s.accessTokenChecker.getExpectedClaims(conf)
	if err != nil {
		return errors.New("Failed to get expected claims from config file")
	}

	loggers.ConfigLogger().Comment("accessTokenChecker: %#v", s.accessTokenChecker)

	return nil
}

func (s *NFServer) maphostConfig() (err error) {
	var client uclient.HTTP
	var scheme string

	c := s.cfg.GetConfig("nf-server.scheme")
	if c != nil {
		scheme = c.GetString("sigtran", "http")
	} else {
		return errors.New("Need to Config HTTP scheme")
	}

	if scheme == "http" || scheme == "https" {
		client = s.httpcli
	} else if scheme == "h2c" {
		client = s.http2c
	} else {
		return errors.New("Invalid sigtran schme")
	}

	mapdomain := os.Getenv("MAP_POD_DOMAIN")
	addrs, err := s.resolver.ResolveService(context.Background(), mapdomain)
	if err != nil {
		return err
	}

	mapport := os.Getenv("MAP_POD_DOMAIN_PORT")
	for k, v := range addrs {
		addrs[k] = v + mapport
		loggers.ConfigLogger().Comment("MAP-POD[%d] = %s", k, addrs[k])
	}

	maphost := os.Getenv("MAP_POD_HOST")
	if maphost != "" {
		s.routermap = uclient.HTTPRouter{
			Scheme: scheme,
			//			Servers:        []string{addrs[0] + mapport, addrs[1] + mapport},
			Servers:        addrs,
			Client:         client,
			CircuitBreaker: s.circuitBreaker,
			//CircuitBreaker:   nil,
			Random: true, //act-act 구조에 따른 라우팅을 위해.......
			//RetryStatusCodes: uclient.StatusCodeSet(400, 403, 404, 502, 508),
		}
	} else {
		return errors.New("Set Fail : maphost Config")
	}

	return nil

}

func (s *NFServer) diahostConfig() (err error) {
	var scheme string
	var client uclient.HTTP

	c := s.cfg.GetConfig("nf-server.scheme")
	if c != nil {
		scheme = c.GetString("diameter", "http")
	} else {
		return errors.New("Need to Config HTTP scheme")
	}

	if scheme == "http" || scheme == "https" {
		client = s.httpcli
	} else if scheme == "h2c" {
		client = s.http2c
	} else {
		return errors.New("Invalid diameter schme")
	}
	diahost := os.Getenv("DIA_POD_HOST")

	if diahost != "" {
		s.routerdia = uclient.HTTPRouter{
			Scheme:  scheme,
			Servers: []string{diahost, diahost},
			Client:  client,
			//CircuitBreaker:   s.circuitBreaker,
			Random: false,
			//RetryStatusCodes: uclient.StatusCodeSet(400, 403, 404, 502, 508),

		}
	} else {
		return errors.New("Set Fail : maphost Config")
	}

	return nil

}

func (s *NFServer) InitUdmCli(supi string) error {
	var err error

	s.udmCli, err = s.scpClient.OpenToNfTypeDyn(context.Background(),
		"UDM",
		scpcli.ClientOption{
			Versions: utypes.Labels{
				"nudm-sdm":  "v1",
				"nudm-uecm": "v1",
			},
			StaticParams: utypes.Map{
				"supi": supi,
			},
			Persistence: true,
			AutoClose:   true,
		},
		time.Minute*10,
	)
	if err != nil {

		amfSupi := "amf-" + supi
		loggers.InitLogger().Comment("Delete DB(Fail NRF connection), %s, %s", amfSupi, amfSupi)
		rval, _ := s.redisDao.GetSubBySUPI(amfSupi)
		if rval == 1 {
			s.redisDao.DelSub(amfSupi)
		}
		rval, _ = s.mysqlDao.GetSubInfoByKEY(amfSupi)
		if rval == 1 {
			s.mysqlDao.Delete(amfSupi)
		}

		sdmSupi := "sdm-" + supi
		loggers.InitLogger().Comment("Delete DB(Fail NRF connection), %s, %s", sdmSupi, sdmSupi)
		rval, _ = s.redisDao.GetSubBySUPI(sdmSupi)
		if rval == 1 {
			s.redisDao.DelSub(sdmSupi)
		}
		rval, _ = s.mysqlDao.GetSubInfoByKEY(sdmSupi)
		if rval == 1 {
			s.mysqlDao.Delete(sdmSupi)
		}

		return err
	}

	return nil
}

func (s *NFServer) InitEchoHandler(trace utrace.Trace) {
	const NFServerSvcName = "NFServer"
	s.Handler = echo.New()

	if s.traceInfo.OnOff == true {
		s.trace = trace
		s.Handler.Use(svctracemgr.MiddleWare(s.trace, NFServerSvcName))
	}
}

func (s *NFServer) InitHttpSvr(cfg uconf.Config) {
	s.http2SvrCfg = &http2.Server{
		MaxHandlers:                  cfg.GetInt("nf-server.https.max-handler", 0),
		MaxConcurrentStreams:         uint32(cfg.GetInt("nf-server.https.max-concurrent-streams")),
		MaxReadFrameSize:             uint32(cfg.GetInt("nf-server.https.max-readframesize")),
		IdleTimeout:                  cfg.GetDuration("nf-server.https.idle-timeout", 100*time.Second),
		MaxUploadBufferPerConnection: int32(cfg.GetInt("nf-server.https.maxuploadbuffer-per-connection", 65536)),
		MaxUploadBufferPerStream:     int32(cfg.GetInt("nf-server.https.maxuploadbuffer-per-stram", 65536)),
	}

}

func NewNFServer(
	cfg uconf.Config,
	redisdaoSet *dao.RedisDaoSet,
	mysqldaoSet *dao.MysqlDaoSet,
	stats *Stats,
	factory scpcli.ScpClientFactory,
	traceMgr interfaces.TraceMgr, // jaeger
	traceInfo *svctracemgr.TraceSvcPod, // EM 정보가지고 북치고 장구치고해야됨
	trace utrace.Trace, // utrace
	httpcli uclient.HTTP, // http or https Interface
	http2c uclient.H2C, // H2C Interface
	circuitBreaker uclient.HTTPCircuitBreaker, // circuitBreaker 구현 Interface 정의
	resolver uregi.Resolver, //service or pod의 IP를 얻어오기 위한 Interface 정의
	dbmgr *db.DBManager,
) *NFServer {

	var err error

	s := &NFServer{
		redisDao:       redisdaoSet.RedisSubDao,
		mysqlDao:       mysqldaoSet.MySqlSubDao,
		stats:          stats,
		traceMgr:       traceMgr,
		scpClient:      factory,
		traceInfo:      traceInfo,
		cfg:            cfg,
		httpcli:        httpcli,
		http2c:         http2c,
		circuitBreaker: circuitBreaker,
		resolver:       resolver,
		dbmgr:          dbmgr,
		notifyUrlAddr:  cfg.GetConfig("smsf").GetString("smsf-noti-url", "127.0.0.1"),
	}

	err = s.SvcReadConfig(cfg)
	if err != nil {
		loggers.InitLogger().Major("FAIL -> Load Config: %v", err)
		return s
	}

	s.InitEchoHandler(trace)
	s.InitHttpSvr(cfg)

	err = s.SvcHTTPConfig(cfg)
	if err != nil {
		loggers.InitLogger().Major("FAIL -> Set HTTP CONFIG(svc): %v", err)
		return s
	}

	//s.Handler.Any("/:s/:v/:o/:n/:c", s.Handle)
	s.Handler.PUT("/:s/:v/:o/:n", s.Handle)
	s.Handler.DELETE("/:s/:v/:o/:n", s.Handle)
	s.Handler.POST("/:s/:v/:o/:n/:c", s.Handle) //uplink
	s.Handler.POST("/:s/:v/:o/:n", s.Handle)    //Failure-Notify

	return s
}

func (s *NFServer) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "NF-Server start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.EventLogger().Data("Shutting down the NF-Server http://%v", s.Addr)
				//s.controller.Close()
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve NF-Server http://%v: %v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.EventLogger().Data(<-waitchnl)
	}

	if s.httpsServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "NF-Server start https://" + s.httpsAddr
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				loggers.EventLogger().Data("Shutting down the NF-Server https://%v, err:%s", s.httpsAddr, err)
				//s.controller.Close()
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve NF-Server https://%v: %v", s.httpsAddr, err.Error())
				os.Exit(1)
			}
		})
		loggers.EventLogger().Data(<-waitchnl)
	}
}

// (검사하도록 설정되어 있는 경우) 수신한 HTTP 메시지 내 access token을 검사한다.
func (s *NFServer) checkAccessToken(ctx echo.Context, supi string) (err error) {

	if !s.accessTokenChecker.shouldCheck {
		loggers.InfoLogger().Comment("Set to NOT check Access Token: Ignoring token")
		return nil
	}

	// e.g. "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6Ikp..."
	authHeaderValue := ctx.Request().Header.Get("Authorization")

	tmpTokens := strings.Split(authHeaderValue, "Bearer ")

	if len(tmpTokens) < 2 {
		var TokenHeaderError = errors.New("Invalid Token Value")
		loggers.ErrorLogger().Major("No Access Token found in message: %s", supi)
		return TokenHeaderError
	}

	// e.g. "eyJhbGciOiJIUzI1NiIsInR5cCI6Ikp..."
	accessToken := tmpTokens[1]

	parsedToken, err := jwt.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {

		var verificationKey interface{}

		if s.accessTokenChecker.signingMethod == "RSA" || s.accessTokenChecker.signingMethod == "RSAPSS" {
			publicKey, err := x509.ParsePKCS1PublicKey(s.accessTokenChecker.key)
			if err != nil {
				loggers.ErrorLogger().Major("%v", err)
				return nil, err
			}

			verificationKey = publicKey

		} else if s.accessTokenChecker.signingMethod == "ECDSA" {
			publicKey, err := jwt.ParseECPublicKeyFromPEM(s.accessTokenChecker.key)
			if err != nil {
				loggers.ErrorLogger().Major("%v", err)
				return nil, err
			}

			verificationKey = publicKey

		} else {
			// HMAC
			verificationKey = s.accessTokenChecker.key
		}

		return verificationKey, nil
	})
	if err != nil {
		loggers.ErrorLogger().Major("JWT parsing error: %v", err)
		return err
	}

	claims := parsedToken.Claims.(jwt.MapClaims)
	loggers.InfoLogger().Data("claims: %#v", claims)

	expectedScope := s.accessTokenChecker.expectedClaims["scope"]
	scope := claims["scope"]
	if scope != expectedScope {
		err = fmt.Errorf("Invalid scope %#v (expected: %#v)", scope, expectedScope)
		loggers.ErrorLogger().Major(err.Error())
		return err
	}

	loggers.InfoLogger().Comment("Access Token is valid")

	return err
}

func (s *NFServer) Handle(ctx echo.Context) error {

	var err error
	var command string

	service := ctx.Param("s")
	version := ctx.Param("v")
	operation := ctx.Param("o")
	supi := ctx.Param("n")

	if ctx.Request().Method == "POST" {
		command = ctx.Param("c")
	}

	//세션이 끊긴 경우, ContentLength가 존재, 이경우 마지막에 세션을 닫아주기 위해 추가
	//이 경우처리 될지 의문이긴 하지만,, 뭐,, 안되면 말고...
	if ctx.Request().Body != nil || ctx.Request().ContentLength > 0 {
		defer ctx.Request().Body.Close()
	}

	switch service {
	case "nsmsf-sms":
		if version != "v1" || operation != "ue-contexts" || len(supi) == 0 {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			s.stats.IncErrCounter(InvalidHttpMsg)
			return err
		}

		switch ctx.Request().Method {
		case "PUT":
			err = s.HandleActivate(ctx, supi)
		case "DELETE":
			err = s.HandleDeactivate(ctx, supi)
		case "POST":
			if command == "sendsms" {
				err = s.HandleUplink(ctx, supi)
			} else {
				s.stats.IncErrCounter(InvalidHttpMsg)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

				s.stats.IncErrCounter(InvalidHttpMsg)
			}
		default:
			loggers.ErrorLogger().Major("Unsupported Request Method : %s", ctx.Request().Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

			s.stats.IncErrCounter(InvalidHttpMsg)
		}
	case "namf-svc":
		if version != "v1" || operation != "sms-failure-notify" || len(supi) == 0 {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			s.stats.IncErrCounter(InvalidHttpMsg)

			return err
		}
		if ctx.Request().Method == "POST" {
			if operation == "sms-failure-notify" {
				err = s.HandleFailureNoti(ctx, supi)
			} else {
				loggers.ErrorLogger().Major("Unsupported operation : %s ", operation)
				s.stats.IncErrCounter(InvalidHttpMsg)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			}
		} else {
			loggers.ErrorLogger().Major("Unsupported Request Method : %s", ctx.Request().Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

			s.stats.IncErrCounter(InvalidHttpMsg)
		}
	case "nudm-svc":
		if version != "v2" || operation != "sdm-change-notify" || len(supi) == 0 {
			loggers.ErrorLogger().Major("Unsupported Request ver : %s or operation : %s", version, operation)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			s.stats.IncErrCounter(InvalidHttpMsg)

			return err
		}
		if ctx.Request().Method == "POST" {
			if operation == "sdm-change-notify" {
				err = s.HandleSdmNotify(ctx, supi)
			} else {
				loggers.ErrorLogger().Major("Unsupported operation : %s ", operation)
				s.stats.IncErrCounter(InvalidHttpMsg)
				err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))
			}
		} else {
			loggers.ErrorLogger().Major("Unsupported Request Method : %s", ctx.Request().Method)
			err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

			s.stats.IncErrCounter(InvalidHttpMsg)
		}
	default:
		loggers.ErrorLogger().Major("Unsupported Service : %s", service)
		err = s.RespondSystemError(ctx, errcode.NotFound(ctx.Request().URL.Path))

		s.stats.IncErrCounter(InvalidHttpMsg)

	}

	if err != nil {
		loggers.ErrorLogger().Major("NF Server Error : %s", err.Error())
	}

	return err
}

func (s *NFServer) HandleActivate(ctx echo.Context, supi string) (err error) {

	//	var accessType string
	//	var uecm, sdm msg5g.Nfinterface
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	smsContext := new(msg5g.UeSmsContextData)
	//	var mutex = &sync.Mutex{}
	s.stats.IncActCounter(ActivateTotal)

	loggers.EventLogger().Data("Activate service start, USER:%s", supi)
	callLogger.Info("RECV ACTIVATE FROM AMF(%s), USER=%s", ctx.RealIP(), supi)

	rawData := ctx.Request().Body

	body, err := ioutil.ReadAll(rawData)

	err = json.Unmarshal(body, &smsContext)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(noBody) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND ACTIVATE RESP TO AMF(DATA), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, ActivateStat)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

	loggers.InfoLogger().Comment("body : %#v", smsContext)

	err = s.checkAccessToken(ctx, supi)
	if err != nil {
		s.stats.IncErrCounter(InvalidToken)
		loggers.ErrorLogger().Major("Activate fail(Get Token) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND ACTIVATE RESP TO AMF(TOKEN), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

	// Check ContentType func
	err = s.checkContentType(ctx)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(%v) : SMSF ----> AMF ==> RespCode :(%s) %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND ACTIVATE RESP TO AMF(%v), RESP:%d, USER=%s", err, http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, ActivateStat)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

	/* check and insert subscriber in redis */
	amfSupi := "amf-" + supi
	val, err := s.checkSubscriber(supi, amfSupi, body)
	if val == true {
		if err != nil {
			loggers.ErrorLogger().Major("Activate fail(redis_insert) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
			callLogger.Info("SEND ACTIVATE RESP TO AMF(DB), RESP:%d, USER=%s", http.StatusForbidden, supi)
			s.StatHttpRespCode(http.StatusForbidden, ActivateStat)
			return s.RespondForbidden(ctx, fmt.Sprintf("%s", err.Error()))
		} else {
			s.StatHttpRespCode(http.StatusNoContent, ActivateStat)
			loggers.InfoLogger().Comment("Already User For Activate in DB, SMSF ----> AMF ===> RespCode :(%s) %d", supi, http.StatusNoContent)
			callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusNoContent, supi)
			ctx.String(http.StatusNoContent, "UE Context for SMS is updated in SMSF")
			return nil
		}
	}

	err = s.InsertSubsCriber(supi, amfSupi, body)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(redis_insert) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND ACTIVATE RESP TO AMF(DB), RESP:%d, USER=%s", http.StatusForbidden, supi)
		s.StatHttpRespCode(http.StatusForbidden, ActivateStat)
		return s.RespondForbidden(ctx, fmt.Sprintf("%s", err.Error()))
	}

	err = s.InitUdmCli(supi)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(Set NRF Connection) : SMSF ----> AMF ==> RespCode :(%s) %d Reason:%v", supi, http.StatusForbidden, err)
		callLogger.Info("SEND ACTIVATE RESP TO AMF(NRF DISCOVERY), RESP:%d, USER=%s", http.StatusForbidden, supi)
		return s.RespondForbidden(ctx, "NRF Connect Fail")
	}

	err = s.Uecm_reig(supi, amfSupi, smsContext)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(UECM Regi), Send Resp : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusForbidden)
		callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusForbidden, supi)
		s.redisDao.DelSub(amfSupi)
		s.mysqlDao.Delete(amfSupi)
		return s.RespondForbidden(ctx, "Service Not Allowed")
	}

	respCode, err, sdmRespData, sdmResp := s.Sdm_Get(supi, smsContext)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(SDM-GET) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusForbidden)
		callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusForbidden, supi)
		rval := s.redisDao.DelSub(amfSupi)
		if rval == -1 {
			loggers.ErrorLogger().Major("redis_delete fail(), AMF user:%s", supi)
		} else {
			s.mysqlDao.Delete(amfSupi)
		}
		return s.RespondForbidden(ctx, "Service Not Allowed")
	} else if respCode == http.StatusNotFound { //StatusNotFound 인 경우
		loggers.ErrorLogger().Major("Activate fail(SDM-GET) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusNotFound)
		callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusNotFound, supi)
		s.StatHttpRespCode(respCode, SdmgetStat)
		return s.RespondNotFound(ctx)
	}

	err = s.Sdm_Subscription(supi, amfSupi, smsContext, sdmRespData, sdmResp, body)
	if err != nil {
		loggers.ErrorLogger().Major("Activate fail(SDM-GET) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusForbidden)
		callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusForbidden, supi)
		s.redisDao.DelSub(amfSupi)
		s.mysqlDao.Delete(amfSupi)
		//	s.redisDao.DelSub(sdmSupi)
		//	s.mysqlDao.Delete(sdmSupi)
		return s.RespondForbidden(ctx, "Service Not Allowed")
	} else {

		loggers.InfoLogger().Comment("Activate Response To AMF : %d", http.StatusCreated)
		callLogger.Info("SEND ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusCreated, supi)
		s.StatHttpRespCode(http.StatusCreated, ActivateStat)

		//ctx.JSONBlob(http.StatusCreated, activateBody)
		ctx.JSONBlob(http.StatusCreated, body)
	}

	return nil
}

func (s *NFServer) HandleDeactivate(ctx echo.Context, supi string) (err error) {

	var rval, chk int
	var amfSub, sdmSub []byte
	var uecmURL string
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	loggers.InfoLogger().Comment("Deactivate service start, USER:%s", supi)
	callLogger.Info("RECV DE-ACTIVATE FROM AMF(%s), USER=%s", ctx.RealIP(), supi)

	err = s.InitUdmCli(supi)
	if err != nil {
		loggers.ErrorLogger().Major("Dectivate fail(NRF Clinet) : SMSF ----> AMF ==> RespCode :(%s) %d, Reason:%v", supi, http.StatusForbidden, err)
		callLogger.Info("SEND DE-ACTIVATE RESP TO AMF(NRF), RESP:%d, USER=%s", http.StatusForbidden, supi)
		return s.RespondForbidden(ctx, "NRF Connect Fail")
	}

	smsContext := msg5g.UeSmsContextData{}
	subInfo := msg5g.SubInfo{}

	//	accept := ctx.Request().Header.Get("accept") // 임시로 막음

	s.stats.IncDeactCounter(DeactivateTotal)
	/*
		if len(accept) == 0 {
			loggers.ErrorLogger().Major("HTTP Deactivate Msg, Missing parameter")
			s.StatHttpRespCode(http.StatusBadRequest, DeactivateStat)

			return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		} //임시로 막음
	*/

	err = s.checkAccessToken(ctx, supi)
	if err != nil {
		loggers.ErrorLogger().Major("Dectivate fail(Get Token) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND DE-ACTIVATE RESP TO AMF(TOKEN), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		s.stats.IncErrCounter(InvalidToken)
		return err
	}

	amfSupi := "amf-" + supi
	rval, amfSub = s.mysqlDao.GetSubInfoByKEY(amfSupi)
	if rval == 1 {
		_ = json.Unmarshal(amfSub, &smsContext)
		//s.redisDao.DelSub(amfSupi)
		s.mysqlDao.Delete(amfSupi)
	} else {
		chk++
	}

	sdmSupi := "sdm-" + supi
	rval, sdmSub = s.mysqlDao.GetSubInfoByKEY(sdmSupi)
	if rval == 1 {
		_ = json.Unmarshal(sdmSub, &subInfo)
		//s.redisDao.DelSub(sdmSupi)
		s.mysqlDao.Delete(sdmSupi)
	} else {
		chk++
	}

	if chk > 0 {
		loggers.InfoLogger().Comment("Get fail Sub in AMF, USER : %s", supi)
		//	ctx.String(http.StatusNotFound, "UE Context for SMS is updated in SMSF")
		s.StatHttpRespCode(http.StatusNotFound, DeactivateStat)
		loggers.ErrorLogger().Major("Dectivate fail(NotFound User) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusNotFound)
		callLogger.Info("SEND DE-ACTIVATE RESP TO AMF(NO-SUB), RESP:%d, USER=%s", http.StatusNotFound, supi)
		return s.RespondNotFound(ctx)
	}

	// publish logic to delete sub in each pod's redis
	s.dbmgr.Publish(supi)

	hdr := http.Header{}
	hdr.Add("accept", "*/*")

	if len(subInfo.SubscriptionID) == 0 {
		loggers.ErrorLogger().Major("Don't Get Subscribtion ID, USER=%s", supi)
	} else {

		// Unsubscribe process
		SubsContext := context.Background()
		if s.traceInfo.OnOff == true {
			SubsContext = uclient.ContextWithTraceLabel(SubsContext, "supi", supi)
		}

		// Make SDM_Subscription Request URL
		scrURL := fmt.Sprintf("/%s/sdm-subscriptions/%s", supi, subInfo.SubscriptionID)

		// Send Request UDM_UECM
		res, err := s.udmCli.ServiceRequest(
			SubsContext,
			"nudm-sdm",
			utypes.Map{
				"supi":             smsContext.Supi,
				"udmGroupId":       smsContext.UdmGroupId,
				"gpsi":             smsContext.Gpsi,
				"routingIndicator": smsContext.RoutingIndicator,
			},
			"DELETE",
			scrURL,
			hdr,
			"",
			time.Second*10,
		)

		if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
			loggers.ErrorLogger().Major("Send Fail Nudm-SDM(SDM-Subscriptions) DELETE err : %s, USER:%s", err, supi)
			callLogger.Info("SEND FAIL SDM-SUBSCRIPTION(DELETE) TO UDM, ERR:%s, USER=%s", err, supi)
			s.stats.IncErrCounter(InvalidSDMSUBSSendHttpMsg)
		} else {
			loggers.InfoLogger().Comment("Send Succ Nudm-SDM(SDM-Subscriptions) DELETE, USER:%s", supi)
			callLogger.Info("SEND SUCC SDM-SUBSCRIPTION(DELETE) TO UDM, USER=%s", supi)
		}

		if res.StatusCode() >= 300 {
			loggers.ErrorLogger().Minor("Recv Nudm-SDM(SDM-Subscriptions) Resp(NACK) Result Code : %d(%s), USER:%s", res.StatusCode(), res.ResponseString(), supi)
		} else {
			loggers.InfoLogger().Comment("Recv Nudm-SDM(SDM-Subscriptions) Resp(ACK) Result Code : %d , USER:%s", res.StatusCode(), supi)
		}
	}

	// Deregistration Process
	if smsContext.AccessType == "3GPP_ACCESS" {
		uecmURL = fmt.Sprintf("/%s/registrations/smsf-3gpp-access", supi)
	} else {
		uecmURL = fmt.Sprintf("/%s/registrations/smsf-non-3gpp-access", supi)
	}

	deactContext := context.Background()
	if s.traceInfo.OnOff == true {
		deactContext = uclient.ContextWithTraceLabel(deactContext, "supi", supi)
	}
	// Request UDM_UECM(Deactivate)
	d_res, d_err := s.udmCli.ServiceRequest(
		deactContext,
		"nudm-uecm",
		utypes.Map{
			"supi":             smsContext.Supi,
			"udmGroupId":       smsContext.UdmGroupId,
			"gpsi":             smsContext.Gpsi,
			"routingIndicator": smsContext.RoutingIndicator,
		},
		"DELETE",
		uecmURL,
		hdr,
		"",
		time.Second*10,
	)

	if d_err != nil {
		loggers.ErrorLogger().Major("Send Fail Nudm-UECM(SMSF De-Registration) DELETE err : %s, USER:%s", d_err, supi)
		callLogger.Info("SEND FAIL DE-REGI TO UDM, ERR:%s, USER=%s", d_err, supi)
	} else {
		loggers.InfoLogger().Comment("Send Succ Nudm-UECM(SMSF De-Registration) DELETE, USER:%s", supi)
		callLogger.Info("SEND SUCC DE-REGI TO UDM, USER=%s", supi)
	}

	if d_res.StatusCode() >= 300 {
		s.stats.IncDeregCounter(DeregTotal)
		s.StatHttpRespCode(d_res.StatusCode(), DeregStat)
		loggers.ErrorLogger().Minor("Recv Nudm-EUCM(SMSF De-Registration) Resp(NACK) Result Code : %d(%s), USER:%s", d_res.StatusCode(), d_res.ResponseString(), supi)
	} else {
		loggers.InfoLogger().Comment("Recv Nudm-EUCM(SMSF De-Registration) Resp(ACK), Result Code : %d, USER:%s", d_res.StatusCode(), supi)
		s.stats.IncDeregCounter(DeregTotal)
		s.StatHttpRespCode(d_res.StatusCode(), DeregStat)
	}
	callLogger.Info("RECV DE-REGI RESP FROM UDM, RESP:%d, USER=%s", d_res.StatusCode(), supi)

	loggers.InfoLogger().Comment("Dectivate Succ : SMSF ----> AMF ==> RespCode:%d USER=%s", http.StatusNoContent, supi)
	callLogger.Info("SEND DE-ACTIVATE RESP TO AMF, RESP:%d, USER=%s", http.StatusNoContent, supi)

	ctx.String(http.StatusNoContent, "UE Context for SMS is updated in SMSF") //204  나중에 삭제...
	s.StatHttpRespCode(http.StatusNoContent, DeactivateStat)

	return nil
}

func (s *NFServer) HandleUplink(ctx echo.Context, supi string) (err error) {

	var bodyOfBinaryPart []byte
	var rpduType byte
	var contentsId string
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var cpdata *cpdu.Cpmessage
	//var mutex = &sync.Mutex{}

	//	accept := ctx.Request().Header.Get("accept") // 임시로 막음
	loggers.InfoLogger().Comment("Uplink service start, USER:%s", supi)
	callLogger.Info("RECV UPLINK FROM AMF(%s), USER=%s", ctx.RealIP(), supi)

	s.stats.IncUplinkCounter(UplinkTotal)

	err = s.checkAccessToken(ctx, supi)
	if err != nil {
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		s.stats.IncErrCounter(InvalidToken)
		loggers.ErrorLogger().Major("Uplink Fail(Get Token) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(TOKEN), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return err
	}

	// Check ContentType func
	contentsType := ctx.Request().Header.Get("Content-Type")
	err = s.checkContentType(ctx)
	if err != nil {
		loggers.ErrorLogger().Major("Uplink Fail(Missing Content-Type) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(HEADER), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	}

	uplinkMsg := msg5g.UplinkSMS{}

	checkBody := ctx.Request().Body
	if checkBody == nil {
		loggers.ErrorLogger().Major("Uplink Fail(Missing Req. Body) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	}

	//var uplinkBody []byte
	uplinkBody, err := ioutil.ReadAll(checkBody)
	if err != nil {
		loggers.ErrorLogger().Major("Uplink Fail(Get Body) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	}
	ctx.Request().Body = ioutil.NopCloser(bytes.NewBuffer(uplinkBody))

	mediaType, params, err := mime.ParseMediaType(contentsType)
	if err != nil {
		loggers.ErrorLogger().Major("Uplink Fail(MIME Parsing Error) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	}

	loggers.InfoLogger().Comment("mediaType : %s, params : %s", mediaType, params)

	if strings.Compare(mediaType, "multipart/related") == 0 {
		bytesBuf := bytes.NewReader(uplinkBody)
		r := related.NewReader(bytesBuf, params)

		part, err := r.NextPart()
		for part != nil && err == nil {
			if part.Root {
				if common.IsContain(part.Header["Content-Type"], "application/json") {
					bodyOfJSONPart, err := ioutil.ReadAll(part)
					if err != nil {
						loggers.ErrorLogger().Major("Read Error JSON Data, USER : %s", supi)
						s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)

						loggers.ErrorLogger().Major("Uplink Fail(Read Json) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
						callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
						return s.RespondBadRequest(ctx)
					}
					loggers.InfoLogger().Comment("JSON contents[supi : %s] : %s", supi, bodyOfJSONPart)
					//mutex.Lock()
					err = json.Unmarshal(bodyOfJSONPart, &uplinkMsg)
					//mutex.Unlock()
					if err != nil {
						loggers.ErrorLogger().Major("JSON unmarshalling Error, err:%s", err)
						s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
						loggers.ErrorLogger().Major("Uplink Fail(Json Unmarshal) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
						callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
						return s.RespondBadRequest(ctx)
					}
					loggers.InfoLogger().Comment("SmsRecordId : %s, gpsi : %s", uplinkMsg.SmsRecordID, uplinkMsg.Gpsi)

				} else {
					loggers.ErrorLogger().Major("Fail to parse Content-Type of the root part : %s, it should be application/json, USER:%s", contentsType, supi)
					s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
					loggers.ErrorLogger().Major("Uplink Fail(Missing Parameter) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
					callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
					return s.RespondBadRequest(ctx)
				}
			} else {
				if common.IsContain(part.Header["Content-Type"], "application/vnd.3gpp.sms") {

					contentsId = part.Header["Content-Id"][0][0:len(part.Header["Content-Id"][0])]
					if len(contentsId) == 0 {
						loggers.ErrorLogger().Major("Does not exist Content-Id, USER:%s", supi)
						s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)

						loggers.ErrorLogger().Major("Uplink Fail(Not exist Content-Id) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
						callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
						return s.RespondBadRequest(ctx)
					} else if contentsId != uplinkMsg.SmsPayloads[0].ContentID {
						loggers.ErrorLogger().Major("binary data header contentId : %s, JSON data contentId : %s", contentsId, uplinkMsg.SmsPayloads[0].ContentID)
						loggers.ErrorLogger().Major("Uplink Fail(Content-Id) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
						callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
						s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
						return s.RespondBadRequest(ctx)
					}

					bodyOfBinaryPart, err = ioutil.ReadAll(part) //binary

					if err != nil {
						loggers.ErrorLogger().Major("Fail to read body of binary part : %s", err)
						s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)

						loggers.ErrorLogger().Major("Uplink Fail(Read body of binary part) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
						callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
						return s.RespondBadRequest(ctx)
					}
					loggers.InfoLogger().Comment("Content-ID : %s", contentsId)
					loggers.InfoLogger().Comment("contents : %x", bodyOfBinaryPart)
				} else {
					loggers.ErrorLogger().Major("Fail to parse Content-Type of the part : %s, it should be application/3gpp.vnd.com", contentsType)
					s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
					loggers.ErrorLogger().Major("Uplink Fail(Read Content-Type of binary part) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
					callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)

					return s.RespondBadRequest(ctx)
				}
			}
			part, err = r.NextPart()
		}
	} else {
		loggers.ErrorLogger().Major("HTTP msg contents type is not  multipart/related, USER:%s", supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		loggers.ErrorLogger().Major("Uplink Fail(Invalid ContentType) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)

		return s.RespondBadRequest(ctx)
	}

	if bodyOfBinaryPart == nil {
		loggers.ErrorLogger().Major("body of binary part is NULL, contentsId:%s, smsRecordId:%s, gpsi:%s", contentsId, uplinkMsg.SmsRecordID, uplinkMsg.Gpsi)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)

		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	} else if len(bodyOfBinaryPart) == 0 {
		loggers.ErrorLogger().Major("body of binary part len is ZERO")
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)

		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)
		return s.RespondBadRequest(ctx)
	}

	cpdata = cpdu.Decoding(bodyOfBinaryPart)

	if cpdata == nil {
		loggers.ErrorLogger().Major("Empty CP-DATA in Uplink, USER:%s", supi)
		s.StatHttpRespCode(http.StatusBadRequest, UplinkStat)

		loggers.ErrorLogger().Major("Uplink Fail(Empty CP-DATA) : SMSF ----> AMF ==> RespCode :(%s) %d", supi, http.StatusBadRequest)
		callLogger.Info("SEND UPLINK RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondBadRequest(ctx)
	}

	if cpdata.MessageType == cpdu.TypeCpData {

		callLogger.Info("UPLINK MSG IS MO(CP-DATA:SUBMIT or DELIVERY-REPORT), USER=%s", supi)

		sdmSupi := "sdm-" + supi

		rval, rdata := s.redisDao.GetSubBySUPI(sdmSupi)
		if rval == -1 {
			rval, rdata = s.mysqlDao.GetSubInfoByKEY(sdmSupi)
			if rval == -1 {
				s.StatHttpRespCode(http.StatusNotFound, UplinkStat)

				loggers.ErrorLogger().Major("Uplink Response Send(Get DB-Data) : SMSF ----> AMF ==> RespCode : %d, SUPI : %s", http.StatusNotFound, supi)
				callLogger.Info("SEND UPLINK RESP TO AMF(DB), RESP:%d, USER=%s", http.StatusNotFound, supi)
				return s.RespondNotFound(ctx)
			}
			s.redisDao.InsSub(sdmSupi, rdata)
		}

		//mutex.Lock()
		sub := msg5g.SubInfo{}
		err = json.Unmarshal(rdata, &sub)
		//mutex.Unlock()

		if sub.MoSmsSubscribed == false {
			loggers.InfoLogger().Comment("Mo SMS sercice doest not subscribed. USER : %s", supi)
			loggers.ErrorLogger().Major("Uplink Response Send(by MoSmsSubscribed) : SMSF ----> AMF ==> RespCode : %d, SUPI:%s", http.StatusNotFound, supi)
			callLogger.Info("SEND UPLINK RESP TO AMF(BARRING), RESP:%d, USER=%s", http.StatusNotFound, supi)
			s.StatHttpRespCode(http.StatusNotFound, UplinkStat)

			return s.RespondNotFound(ctx)
		}

		if sub.MoSmsBarringAll == true {
			loggers.InfoLogger().Comment("Mo SMS sercice was barring. USER : %s", supi)
			loggers.ErrorLogger().Major("Uplink Response Send(by MoSmsBarringAll) : SMSF ----> AMF ==> RespCode : %d, SUPI:%s", http.StatusNotFound, supi)
			callLogger.Info("SEND UPLINK RESP TO AMF(BARRING), RESP:%d, USER=%s", http.StatusNotFound, supi)
			s.StatHttpRespCode(http.StatusNotFound, UplinkStat)

			return s.RespondNotFound(ctx)
		}

		rpdata := rpdu.Decoding(cpdata.CpUserData)
		rpduType = rpdata.RpMessageType

		loggers.InfoLogger().Comment("MSG is CP-DATA, SUPI:%s", supi)
		loggers.InfoLogger().Comment("rpdata : %x, %x", cpdata.CpUserData, rpduType)

		exec.SafeGo(func() {
			s.SendMsgMapIF(cpdata.CpUserData,
				rpduType, supi, uplinkMsg.Gpsi,
				false, "NONE", rpdata.RpError)
			// snedmsgmapif 한 후에.. 실패면 AMF로 뭔가 보내줘야되는거 아닌가? // 보다보니.. 2020-01-06
		})

	} else {

		if cpdata.MessageType == cpdu.TypeCpAck {
			loggers.InfoLogger().Comment("MSG is UPLINK:CP-ACK(%d), SUPI:%s", cpdata.MessageType, supi)
			callLogger.Info("UPLINK MSG IS CP-ACK, USER=%s", supi)
		} else {
			loggers.InfoLogger().Comment("MSG is UPLINK:CP-ERROR(%d), SUPI:%s", cpdata.MessageType, supi)
			callLogger.Info("UPLINK MSG IS CP-ERROR, USER=%s", supi)
		}

		ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		ctx.Response().WriteHeader(http.StatusOK)

		s.StatHttpRespCode(http.StatusOK, UplinkStat)
		loggers.InfoLogger().Comment("Uplink Response Send(CP-ACK/ERROR) : SMSF ----> AMF ==> RespCode : %d, SUPI:%s", http.StatusOK, supi)
		callLogger.Info("SEND UPLINK(CP-ACK/ERROR) RESP TO AMF, RESP:%d, USER=%s", http.StatusOK, supi)

		return json.NewEncoder(ctx.Response()).Encode(msg5g.UplinkResp{
			SmsRecordID:    "smsRecordId" + uplinkMsg.SmsRecordID,
			DeliveryStatus: msg5g.SMS_DELIVERY_COMPLETE,
		})

	}

	// N1N2 Msg(CP-ACK)
	cpAck := MakeCPAck(supi, cpdata)
	exec.SafeGo(func() {
		s.SendN1N2Msg(cpAck, supi)
	})

	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	ctx.Response().WriteHeader(http.StatusOK)
	s.StatHttpRespCode(http.StatusOK, UplinkStat)

	loggers.InfoLogger().Comment("Uplink Response Send(CP-DATA) : SMSF ----> AMF ==> RespCode : %d, SUPI:%s", http.StatusOK, supi)
	callLogger.Info("SEND UPLINK(CP-DATA) RESP TO AMF, RESP:%d, USER=%s", http.StatusOK, supi)

	return json.NewEncoder(ctx.Response()).Encode(msg5g.UplinkResp{
		SmsRecordID:    "smsRecordId" + uplinkMsg.SmsRecordID,
		DeliveryStatus: msg5g.SMS_DELIVERY_COMPLETE,
	})
}

func (s *NFServer) HandleFailureNoti(ctx echo.Context, supi string) (err error) {

	loggers.InfoLogger().Comment("FailureNoti service start, USER:%s", supi)
	callLogger.Info("RECV FAILURE-NOTI FROM AMF(%s), USER=%s", ctx.RealIP(), supi)

	err = s.checkAccessToken(ctx, supi)
	if err != nil {
		callLogger.Info("SEND FAILUER-NOTI RESP TO AMF(TOKEN), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		s.stats.IncErrCounter(InvalidToken)
		return err
	}

	accept := ctx.Request().Header.Get("accept")
	contentsType := ctx.Request().Header.Get("Content-Type")

	loggers.InfoLogger().Comment("name : %s, accept : %s, contents_type : %s", supi, accept, contentsType)
	//	s.stats.IncUplinkCounter(UplinkTotal)

	failNoti := new(msg5g.N1N2MsgTxfrFailureNotification)

	if err = ctx.Bind(failNoti); err != nil {
		loggers.ErrorLogger().Major("Get Data Fail. Nobody, err : %v", err)
		callLogger.Info("SEND FAILUER-NOTI RESP TO AMF(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.ErrorBadRequest(ctx)
	}

	loggers.InfoLogger().Comment("USER : %s, body : %#v", supi, failNoti)

	s.SendMsgMapIF(nil, rpdu.RP_DATA_MS_N, supi, "NONE", true,
		failNoti.Cause, 2)

	ctx.String(http.StatusNoContent, "No Content")
	callLogger.Info("SEND FAILURE-NOTI RESP TO AMF, RESP:%d, USER=%s", http.StatusNoContent, supi)

	return nil
}

func (s *NFServer) SendMsgMapIF(rpmsg []byte,
	msgType byte,
	supi string,
	gpsi string,
	notiCheck bool,
	resultCode string,
	result byte,
) {

	var msg msg5g.Nfinterface
	var smscURL, contentsType string
	var reqBody []byte
	//	var rsp *http.Response
	var rsp *uclient.HTTPResponse
	var err error

	//time.Sleep(100 * time.Millisecond)
	convSupi := strings.TrimLeft(supi, "imsi-")
	convGpsi := strings.TrimLeft(gpsi, "msisdn-")

	contentsType = "application/json"
	if notiCheck == true {
		smscURL = fmt.Sprintf("/map/v1/%s/rpresp", convSupi)
		MsgType := "Failure-Notify"
		msg = msg5g.MtFailNoti{
			MsgType:    MsgType,
			ResultCode: resultCode,
		}
		reqBody, err = msg.Make()
	} else {
		if msgType == rpdu.RP_DATA_MS_N {
			smscURL = fmt.Sprintf("/svc/v1/%s/rpdata", convSupi)
		} else {
			smscURL = fmt.Sprintf("/map/v1/%s/rpresp", convSupi)
		}

		if msgType == rpdu.RP_DATA_MS_N {
			msg = msg5g.MoSMS{
				Rpmsg: rpmsg,
				Gpsi:  convGpsi,
			}
			reqBody, err = msg.Make()
		} else {
			msg = msg5g.MtResp{
				Rpmsg:  rpmsg,
				Result: result,
			}
			reqBody, err = msg.Make()
		}
	}

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", contentsType)

	data := configmgr.DecisionStoragePop(supi)

	if data == nil {
		loggers.ErrorLogger().Major("SIGTRAN OR DIAMETER SELELT FAIL, USER : %s", supi)
		callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s)(PREFIX), USER=%s", smscURL, supi)
		return
	}

	if data.Decision == "SIGTRAN" {
		err = s.maphostConfig() // map-httpif 연동
		if err != nil {
			loggers.ErrorLogger().Major("FAIL -> Set MapHost: %v", err)
			callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
			return
		}

		loggers.InfoLogger().Comment("Send to MAP IF, URL : %s%s", s.routermap.Servers[0], smscURL)

		rsp, err = s.routermap.SendRequest(context.Background(), smscURL, "POST", hdr, reqBody, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("Send Fail SMS Req, err : %s, USER : %s", err, supi)
			callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
			return
		} else {
			loggers.InfoLogger().Comment("Send Succ SMS Req Succ to MAP IF, USER:%s", supi)
			callLogger.Info("SEND SUCC MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
		}
	} else if data.Decision == "DIAMETER" {

		err = s.diahostConfig() // dia-httpif 연동
		if err != nil {
			loggers.ErrorLogger().Major("FAIL -> Set DiaHost: %v", err)
			callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
			return
		}

		loggers.InfoLogger().Comment("Send to DIA IF, URL : %s%s", s.routerdia.Servers[0], smscURL)
		loggers.InfoLogger().Comment("Send Data : %s", string(reqBody))

		rsp, err = s.routerdia.SendRequest(context.Background(), smscURL, "POST", hdr, reqBody, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("Send Fail SMS Req err : %s, USER:%s", err, supi)
			callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
			return
		} else {
			loggers.InfoLogger().Comment("Send Succ SMS Req Succ to DIA IF, USER:%s", supi)
			callLogger.Info("SEND SUCC MSG TO SMSF-IF(%s), USER=%s", smscURL, supi)
		}

	} else {
		loggers.ErrorLogger().Major("Does not find Prefix info, USER:%s", supi)
		callLogger.Info("SEND FAIL MSG TO SMSF-IF(%s)(PREFIX), USER=%s", smscURL, supi)
		return
	}

	callLogger.Info("RECV SMS MSG RESP FROM SMSC-IF, RESP:%d, USER=%s", rsp.StatusCode, supi)

	if rsp.StatusCode > 300 {
		loggers.ErrorLogger().Major("Recv SMS Req Resp(NACK), Result Code : %d, USER:%s", rsp.StatusCode, supi)
		return
	}

	loggers.InfoLogger().Comment("Rec SMS Respone(ACK), Result Code(From IF) : %d, USER:%s", rsp.StatusCode, supi)

	return
}

func (s *NFServer) SendN1N2Msg(cpmsg []byte, supi string) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var msg msg5g.Nfinterface

	loggers.InfoLogger().Comment("Send to AMF N1N2 Msg(CP_ACK), USER:%s", supi)

	amfSupi := "amf-" + supi

	rval, rdata := s.redisDao.GetSubBySUPI(amfSupi)
	if rval == -1 {
		rval, rdata = s.mysqlDao.GetSubInfoByKEY(amfSupi)
		if rval == -1 {

			s.StatHttpRespCode(http.StatusNotFound, N1n2Stat)
			return
		}
		s.redisDao.InsSub(amfSupi, rdata)
	}

	smsContext := msg5g.UeSmsContextData{}
	err := json.Unmarshal(rdata, &smsContext)
	if err != nil {
		loggers.ErrorLogger().Major("json.Unmarshal Err:%s", err)
		return
	}

	amfId := smsContext.AmfId

	cli, err := s.scpClient.OpenToNf(
		context.Background(),
		"AMF",
		amfId,
		scpcli.ClientOption{
			Versions: utypes.Labels{
				"namf-comm": "v1",
				"namf-mt":   "v1",
			},
			Persistence: true,
			AutoClose:   true,
		},
	)

	if err != nil || cli == nil {
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer POST, MO CP-ACK/ERROR) because of NRF interworking, err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL N1N2-TRANSFER(CP-ACK)(NRF) TO AMF, ERR:%s, USER=%s", err, supi)
		return
	}

	loggers.InfoLogger().Comment("Send Succ Namf-COMM(N1N2-Transfer POST, MO CP-ACK/ERROR) because of NRF interworking, err : %s, USER:%s", err, supi)
	defer cli.Close()

	msg = msg5g.N1N2_RESP{cpmsg, false, common.SMSF_BOUNDARY}
	n1n2MsgBody, err := msg.Make()

	n1n2URL := fmt.Sprintf("/ue-contexts/%s/n1-n2-messages", supi)
	contentsType := fmt.Sprintf("multipart/related;boundary=%s", common.SMSF_BOUNDARY)
	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", contentsType)

	N1N2Context := context.Background()
	if s.traceInfo.OnOff == true {
		N1N2Context = uclient.ContextWithTraceLabel(N1N2Context, "supi", supi)
	}

	res, err := cli.ServiceRequest(
		N1N2Context,
		"namf-comm",
		"POST",
		n1n2URL,
		hdr,
		n1n2MsgBody,
		time.Second*10,
	)

	loggers.InfoLogger().Comment("N1N2(svc) Req(CP-ACK), USER:%s, Body : %s", supi, string(n1n2MsgBody))
	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer POST, MO CP-ACK/ERROR) err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL N1N2-TRANSFER(CP-ACK) TO AMF, ERR:%s, USER=%s", err, supi)
		s.stats.IncErrCounter(InvalidN1N2SendHttpMsg)
		return
	}

	loggers.InfoLogger().Comment("Send Succ Namf-COMM(N1N2-Transfer POST, MO CP-ACK/ERROR), USER:%s", supi)
	callLogger.Info("SEND SUCC N1N2-TRANSFER(CP-ACK) TO AMF, USER=%s", supi)

	s.stats.IncN1n2Counter(N1n2Total)

	if res.StatusCode() >= 300 {
		loggers.ErrorLogger().Minor("Recv Namf-COMM(N1N2-Transfer, MO CP-ACK/ERROR) Resp(NACK) Result Code : %d(%s), USER:%s, RespToIF : %d", res.StatusCode(), res.ResponseString(), supi, http.StatusServiceUnavailable)
		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)
	} else {
		loggers.InfoLogger().Comment("Recv Namf-(N1N2-Transfer, MO CP-ACK/ERROR) Resp(ACK), USER:%s, Resp:%d, BODY:%s", supi, res.StatusCode(), res.ResponseString())
		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)
	}

	callLogger.Info("RECV N1N2-TRANSFER(CP-ACK) RESP FROM AMF, RESP:%d, USER=%s", res.StatusCode(), supi)

	return
}

func (s *NFServer) HandleSdmNotify(ctx echo.Context, supi string) (err error) {

	loggers.InfoLogger().Comment("SdmNofity service start, USER:%s", supi)
	callLogger.Info("RECV SDM-NOTIFY FROM UDM(%s), USER=%s", ctx.RealIP(), supi)

	err = s.checkAccessToken(ctx, supi)
	if err != nil {
		callLogger.Info("SEND SDM-NOTIFY RESP TO UDM(TOKEN), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		s.stats.IncErrCounter(InvalidToken)
		return err
	}

	accept := ctx.Request().Header.Get("accept")
	contentsType := ctx.Request().Header.Get("Content-Type")

	loggers.InfoLogger().Comment("name : %s, accept : %s, contents_type : %s", supi, accept, contentsType)

	sdmNoti := new(msg5g.ModificationNotification)

	if err = ctx.Bind(sdmNoti); err != nil {
		loggers.ErrorLogger().Major("Get Data Fail. Nobody, err : %v", err)
		callLogger.Info("SEND SDM-NOTIFY RESP TO UDM(BODY), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.ErrorBadRequest(ctx)
	}

	loggers.InfoLogger().Comment("USER : %s, body : %#v", supi, sdmNoti)

	/* Need to get amf info */
	sdmSupi := "sdm-" + supi

	rval, rdata := s.redisDao.GetSubBySUPI(sdmSupi)
	if rval == -1 {
		rval, rdata = s.mysqlDao.GetSubInfoByKEY(sdmSupi)
		if rval == -1 {
			loggers.InfoLogger().Comment("Does not find subscriber in SDM subscriber. USER : %s", supi)
			loggers.InfoLogger().Comment("Invalid Internal Error : not find Subscriber, supi(%s), respCode(%d)", supi, http.StatusNotFound)
			callLogger.Info("SEND MT-SMS RESP TO SMSF-IF(DB), RESP:%d, USER=%s", http.StatusNotFound, supi)
			return s.RespondNotFound(ctx)
		}
		s.redisDao.InsSub(sdmSupi, rdata)
	}

	sub := msg5g.SubInfo{}
	err = json.Unmarshal(rdata, &sub)

	for _, notis := range sdmNoti.NotifyItems {

		for _, noti := range notis.Changes {
			if noti.Op == "REPLACE" {
				switch noti.Path {
				case "mtSmsSubScribed":
					loggers.InfoLogger().Comment("Changed mtSmsSubScribed, old:%t, new:%t", sub.MtSmsSubscribed, noti.NewValue.(bool))
					sub.MtSmsSubscribed = noti.NewValue.(bool)
				case "mtSmsBarringAll":
					loggers.InfoLogger().Comment("Changed mtSmsBarringAll, old:%t, new:%t", sub.MtSmsSubscribed, noti.NewValue.(bool))
					sub.MtSmsBarringAll = noti.NewValue.(bool)
				case "mtSmsBarringRoaming":
					loggers.InfoLogger().Comment("Changed mtSmsBarringRoaming, old:%t, new:%t", sub.MtSmsSubscribed, noti.NewValue.(bool))
					sub.MtSmsBarringRoaming = noti.NewValue.(bool)
				case "moSmsSubScribed":
					loggers.InfoLogger().Comment("Changed moSmsSubScribed, old:%t, new:%t", sub.MoSmsSubscribed, noti.NewValue.(bool))
					sub.MoSmsSubscribed = noti.NewValue.(bool)
				case "moSmsBarringAll":
					loggers.InfoLogger().Comment("Changed moSmsBarringAll, old:%t, new:%t", sub.MoSmsSubscribed, noti.NewValue.(bool))
					sub.MoSmsBarringAll = noti.NewValue.(bool)
				case "moSmsBarringRoaming":
					loggers.InfoLogger().Comment("Changed moSmsBarringRoaming, old:%t, new:%t", sub.MoSmsSubscribed, noti.NewValue.(bool))
					sub.MoSmsBarringRoaming = noti.NewValue.(bool)
				}
			}
		}
	}

	/* check and insert subscriber in redis */
	s.mysqlDao.Delete(sdmSupi)

	subData, _ := json.Marshal(sub)
	mbody := &dao.MariaInfo{IMSI: sdmSupi, DATA: subData}
	s.mysqlDao.Create(mbody)

	// publish logic to delete sub in each pod's redis
	s.dbmgr.Publish(supi)

	ctx.String(http.StatusNoContent, "No Content")
	callLogger.Info("SEND SDM-NOTIFY RESP TO UDM, RESP:%d, USER=%s", http.StatusNoContent, supi)

	return nil
}
