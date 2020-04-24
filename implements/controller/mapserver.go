package controller

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"net/http"
	"strconv"

	"errors"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/xerrors"

	jsoniter "github.com/json-iterator/go"
	"github.com/philippfranke/multipart-related/related"

	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"

	"camel.uangel.com/ua5g/ulib.git/utrace"
	"camel.uangel.com/ua5g/ulib.git/utypes"
	"camel.uangel.com/ua5g/usmsf.git/implements/svctracemgr"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/tcpmgr"

	"camel.uangel.com/ua5g/usmsf.git/dao"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

const (
	SMSF_ERR   = -1
	SMSF_NOTOK = 0
	SMSF_OK    = 1
)

var cnt int32

type MapServer struct {
	common.HTTPServer
	stats       *Stats
	redisDao    dao.RedisSubDao
	mysqlDao    dao.MySqlSubDao
	traceMgr    interfaces.TraceMgr
	httpServer  *http.Server
	httpsServer *http.Server
	http2SvrCfg *http2.Server
	scpClient   scpcli.ScpClientFactory
	traceInfo   *svctracemgr.TraceSvcPod
	trace       utrace.Trace

	httpcli        uclient.HTTP
	circuitBreaker uclient.HTTPCircuitBreaker
	requires       NFServeriRequires

	connCnt   uint32
	httpsAddr string
	fqdn      string
	nfId      string
	mnc       string
	mcc       string
	isdn      string
	name      string
	realm     string

	notiUrlAddr string
}

const MapServerSvcName = "MapServer"

func (s *MapServer) MapHTTPConfig(cfg uconf.Config) (ret error) {
	s.requires.httpConf = cfg.GetConfig("map-server.http")
	s.requires.httpsConf = cfg.GetConfig("map-server.https")

	if s.requires.httpConf != nil {
		httpAddr := s.requires.httpConf.GetString("address", "")
		httpPort := s.requires.httpConf.GetInt("port", 8080)
		s.Addr = httpAddr + ":" + strconv.Itoa(httpPort)
		httpsvr := &http.Server{
			Addr:    s.Addr,
			Handler: h2c.NewHandler(s.Handler, s.http2SvrCfg),
		}
		http2.VerboseLogs = cfg.GetBoolean("map-server.https.verbose-logs", false)
		http2.ConfigureServer(httpsvr, s.http2SvrCfg)
		s.httpServer = httpsvr
	} else {
		ret = errors.New("Init Fail : HTTP CONFIG(Map)")
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
					loggers.ErrorLogger().Major("Not found TLS configuration (als-server.tls| sepp-tls.internal-ntework| sepp.tls)")
					ret = errors.New("Not found TLS configuration")
					return ret
				}
			}
		}

		certInfo, err := common.NewCertInfoByCfg(tlscfg)
		if err != nil {
			loggers.ErrorLogger().Major("Failed to create CertInfo: error=%#v", err.Error())
			ret = err
			return ret
		}

		s.httpsAddr = httpsAddr + ":" + strconv.Itoa(httpsPort)
		httpssvr := &http.Server{
			Addr:              s.httpsAddr,
			Handler:           s.Handler,
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			TLSConfig:         certInfo.GetServerTLSConfig(),
		}
		http2.VerboseLogs = cfg.GetBoolean("map-server.https.verbose-logs", false)
		http2.ConfigureServer(httpssvr, s.http2SvrCfg)
		s.httpsServer = httpssvr
	} else {
		ret = errors.New("Init Fail : HTTPS CONFIG(Map)")
		return ret
	}

	return nil

}

func (s *MapServer) MapReadConfg(cfg uconf.Config) (ret error) {

	smsfConf := cfg.GetConfig("smsf")

	if smsfConf != nil {

		s.fqdn = smsfConf.GetString("my-fqdn", "smsf.uangel.com")
		s.nfId = smsfConf.GetString("my-nf-id", "abcde-1234-5678-abcd-ef120-1111")
		s.mnc = smsfConf.GetString("mnc", "450")
		s.mcc = smsfConf.GetString("mcc", "06")
		s.isdn = smsfConf.GetString("my-map-id", "821045001234")
		s.name = smsfConf.GetString("my-diameter-name", "smsf")
		s.realm = smsfConf.GetString("my-diameter-realm", "smsf.uangel.com")
	} else {
		ret = errors.New("Init Fail : Config Load(MapReadConfig)")
		return
	}

	return nil

}

func (s *MapServer) InitEchoHandler(trace utrace.Trace) {
	s.Handler = echo.New()

	if s.traceInfo.OnOff == true {
		s.trace = trace
		s.Handler.Use(svctracemgr.MiddleWare(s.trace, MapServerSvcName))
		//	s.Handler.Use(traceMgr.GinHTTPTraceHandler(NFServerSvcName, ulog.InfoLevel))
	}

}

func (s *MapServer) InitHttpMap(cfg uconf.Config) {
	s.http2SvrCfg = &http2.Server{
		MaxHandlers:                  cfg.GetInt("map-server.https.max-handler", 0),
		MaxConcurrentStreams:         uint32(cfg.GetInt("map-server.https.max-concurrent-streams", 5000)),
		MaxReadFrameSize:             uint32(cfg.GetInt("map-server.https.max-readframesize")),
		IdleTimeout:                  cfg.GetDuration("map-server.https.idle-timeout", 100),
		MaxUploadBufferPerConnection: int32(cfg.GetInt("map-server.https.maxuploadbuffer-per-connection", 65536)),
		MaxUploadBufferPerStream:     int32(cfg.GetInt("map-server.https.maxuploadbuffer-per-stram", 65536)),
	}

}

func NewMapServer(cfg uconf.Config,
	redisdaoSet *dao.RedisDaoSet,
	mysqldaoSet *dao.MysqlDaoSet,
	stats *Stats,
	factory scpcli.ScpClientFactory,
	traceMgr interfaces.TraceMgr,
	traceInfo *svctracemgr.TraceSvcPod,
	trace utrace.Trace,
	httpcli uclient.HTTP,
	circuitBreaker uclient.HTTPCircuitBreaker,
) *MapServer {

	var err error

	s := &MapServer{
		redisDao:       redisdaoSet.RedisSubDao,
		mysqlDao:       mysqldaoSet.MySqlSubDao,
		stats:          stats,
		traceMgr:       traceMgr,
		scpClient:      factory,
		traceInfo:      traceInfo,
		httpcli:        httpcli,
		circuitBreaker: circuitBreaker,
		notiUrlAddr:    cfg.GetConfig("smsf").GetString("smsf-noti-url", "127.0.0.1"),
	}

	err = s.MapReadConfg(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL -> Load Config: %v", err)
		return nil
	}

	s.InitEchoHandler(trace)
	s.InitHttpMap(cfg)

	err = s.MapHTTPConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL -> Set HTTP CONFIG(map): %v", err)
		return nil
	}

	s.Handler.POST("/:s/:v/:n/:c", s.Handle)

	return s
}

func (s *MapServer) Start() {
	waitchnl := make(chan string)
	if s.httpServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "NF-Server start http://" + s.Addr
			err := s.httpServer.ListenAndServe()
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the NF-Server http://%v", s.Addr)
				//s.controller.Close()
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve NF-Server http://%v: error=%#v", s.Addr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}

	if s.httpsServer != nil {
		exec.SafeGo(func() {
			waitchnl <- "NF-Server start https://" + s.httpsAddr
			err := s.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				loggers.InfoLogger().Comment("Shutting down the NF-Server https://%v", s.httpsAddr)
				//s.controller.Close()
			} else {
				loggers.ErrorLogger().Critical("Failed to listen and serve NF-Server https://%v: error=%#v", s.httpsAddr, err.Error())
				os.Exit(1)
			}
		})
		loggers.InfoLogger().Comment(<-waitchnl)
	}
}

func (s *MapServer) Handle(ctx echo.Context) error {

	var err error
	var command string

	//	timer := s.stats.StartTranscTimer(StatMapServerTransc)
	//	defer EndTransacTimer(timer, &err)

	service := ctx.Param("s")
	version := ctx.Param("v")
	oSupi := ctx.Param("n")
	msgtype := ctx.Param("c")

	if ctx.Request().Body != nil {
		defer ctx.Request().Body.Close()
	}

	if version != "v1" {
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		loggers.ErrorLogger().Major("Invalid Internal Error : Version(%s), supi(%s), respCode(%d)", version, oSupi, http.StatusBadRequest)
		return err
	}

	supi := fmt.Sprintf("imsi-%s", oSupi)

	switch service {
	case "svc":
		if msgtype != "rpresp" {
			loggers.ErrorLogger().Major("Invalid Internal Error : command(%s), supi(%s), respCode(%d)", command, supi, http.StatusBadRequest)
			err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
		} else {
			err = s.HandleMoResp(ctx, supi)
		}
	case "map":
		if msgtype != "rpdata" {
			loggers.ErrorLogger().Major("Invalid Internal Error : command(%s), supi(%s), respCode(%d)", command, supi, http.StatusBadRequest)
			err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))

		} else {
			err = s.HandleMtService(ctx, supi)
		}
	default:
		loggers.ErrorLogger().Major("Invalid Internal Error : service(%s), supi(%s), respCode(%d)", service, supi, http.StatusBadRequest)
		err = s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))

	}

	if err != nil {
		loggers.ErrorLogger().Major("Map Server Error : %s", err.Error())
	}

	return err
}

func (s *MapServer) HandleMtService(ctx echo.Context, supi string) (err error) {

	var ueRe, n1n2 msg5g.Nfinterface
	//var mutex = &sync.Mutex{}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	//	cType := ctx.Request().URL.Query().Get("Content-Type")

	loggers.InfoLogger().Comment("Receive from Map Service Pod(MT), USER : %s", supi)
	callLogger.Info("RECV MT-SMS FROM SMSF-IF, USER=%s", supi)

	ReqData := new(tcpmgr.HttpIfMtMsg)

	if err = ctx.Bind(ReqData); err != nil {
		loggers.ErrorLogger().Major("Invalid Internal Error : No-Body, supi(%s), respCode(%d)", supi, http.StatusBadRequest)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF(DATA), RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

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

	//mutex.Lock()
	//	sub := msg5g.SubInfo{}
	sub := msg5g.SubInfo{}
	err = json.Unmarshal(rdata, &sub)
	//mutex.Unlock()

	if sub.MtSmsSubscribed == false {
		loggers.InfoLogger().Comment("Mt SMS sercice doest not subscribed. USER : %s", supi)
		loggers.InfoLogger().Comment("Invalid Internal Error : MtSmsSubscribed, supi(%s), respCode(%d)", supi, http.StatusMethodNotAllowed)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF(BARRING), RESP:%d, USER=%s", http.StatusMethodNotAllowed, supi)
		return s.RespondNotAllowed(ctx, "Service Not Allowed")
	}

	if sub.MtSmsBarringAll == true {
		loggers.InfoLogger().Comment("Mt SMS sercice was barring. USER : %s", supi)
		loggers.InfoLogger().Comment("Invalid Internal Error : MtSmsBarringAll, supi(%s), respCode(%d)", supi, http.StatusMethodNotAllowed)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF(BARRING), RESP:%d, USER=%s", http.StatusMethodNotAllowed, supi)
		return s.RespondNotAllowed(ctx, "Service Not Allowed")
	}

	amfSupi := "amf-" + supi
	rval, rdata = s.redisDao.GetSubBySUPI(amfSupi)
	if rval == -1 {
		rval, rdata = s.mysqlDao.GetSubInfoByKEY(amfSupi)
		if rval == -1 {
			loggers.InfoLogger().Comment("Dose not find subscriber in AMF subscriber. USER : %s", supi)
			loggers.InfoLogger().Comment("Invalid Internal Error : not find subscriber, supi(%s), respCode(%d)", supi, http.StatusNotFound)
			callLogger.Info("SEND MT-SMS RESP TO SMSF-IF(DB), RESP:%d, USER=%s", http.StatusNotFound, supi)
			return s.RespondNotFound(ctx)
		}
		s.redisDao.InsSub(amfSupi, rdata)
	}

	//mutex.Lock()
	smsContext := msg5g.UeSmsContextData{}
	err = json.Unmarshal(rdata, &smsContext)
	//mutex.Unlock()

	/* UeReachableEnable */
	amfURL := fmt.Sprintf("/ue-contexts/%s/ue-reachind", supi)

	ueRe = msg5g.UeReach{msg5g.REACHABLE}
	ueReachableBody, err := ueRe.Make()

	cli, err := s.scpClient.OpenToNf(
		context.Background(),
		"AMF",
		smsContext.AmfId,
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
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer, MT) because of NRF interworking, err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL MT-UE-REACH TO AMF(NRF), ERR:%s, USER=%s", err, supi)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondBadRequest(ctx)
	}

	loggers.InfoLogger().Comment("MT Reach Info, AMF's nfId:%s, USER : %s", smsContext.AmfId, supi)

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	ReachContext := context.Background()
	if s.traceInfo.OnOff == true {
		ReachContext = uclient.ContextWithTraceLabel(ReachContext, "supi", supi)
	}
	res, err := cli.ServiceRequest(
		ReachContext,
		"namf-mt",
		"PUT",
		amfURL,
		hdr,
		ueReachableBody,
		time.Second*10,
	)

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Namf-MT(Enable-UE-Reachability) ERR : %s, USER:%s, RespToIF:%d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL MT-UE-REACH TO AMF, ERR:%s, USER=%s", err, supi)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.stats.IncErrCounter(InvalidReachSendHttpMsg)
		return s.RespondBadRequest(ctx)
	} else {
		loggers.InfoLogger().Comment("Send Succ Namf-MT(Enable-UE-Reachabiliy), USER:%s", supi)
		callLogger.Info("SEND SUCC MT-UE-REACH TO AMF, USER=%s", supi)
	}

	s.stats.IncReachCounter(ReachTotal)
	callLogger.Info("RECV MT-UE-REACH RESP FROM AMF, RESP:%d, USER=%s", res.StatusCode(), supi)

	if res.StatusCode() >= 300 {

		loggers.ErrorLogger().Minor("Recv Namf-MT(Enable-UE-Reachability) Resp(NACK) Result Code : %d(%s), USER:%s, RespToIF:%d",
			res.StatusCode(), res.ResponseString(), supi, http.StatusServiceUnavailable)

		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusServiceUnavailable, supi)
		s.StatHttpRespCode(res.StatusCode(), ReachStat)
		return s.RespondServiceUnavailable(ctx, "Service Unavailable")
	} else {
		s.StatHttpRespCode(res.StatusCode(), ReachStat)
	}

	/* N1N2 Msg for MT */
	cpData := MakeCPData(ReqData.RpData)
	loggers.InfoLogger().Comment("Mt-Message cp-data : %X", cpData)

	n1n2URL := fmt.Sprintf("/ue-contexts/%s/n1-n2-messages", supi)
	contentsType := fmt.Sprintf("multipart/related;boundary=%s", common.SMSF_BOUNDARY)

	n1n2 = msg5g.N1N2_MT{cpData, ReqData.Mms, s.notiUrlAddr, supi, common.SMSF_BOUNDARY}
	n1n2MsgBody, err := n1n2.Make()

	hdr = http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", contentsType)

	N1N2Context := context.Background()
	if s.traceInfo.OnOff == true {
		N1N2Context = uclient.ContextWithTraceLabel(N1N2Context, "supi", supi)
	}
	res, err = cli.ServiceRequest(
		N1N2Context,
		"namf-comm",
		"POST",
		n1n2URL,
		hdr,
		n1n2MsgBody,
		time.Second*10,
	)

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer POST, MT) err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL N1N2-TRANSFER(CP-DATA, MT) TO AMF, ERR:%s, USER=%s", err, supi)
		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.stats.IncErrCounter(InvalidN1N2SendHttpMsg)
		return s.RespondBadRequest(ctx)
	} else {
		loggers.InfoLogger().Comment("Send Succ Namf-COMM(N1N2-Transfer POST, MT), USER:%s", supi)
		callLogger.Info("SEND SUCC N1N2-TRANSFER(CP-DATA, MT) TO AMF, USER=%s", supi)
	}

	s.stats.IncN1n2Counter(N1n2Total)
	callLogger.Info("RECV N1N2-TRANSFER(CP-DATA, MT) RESP FROM AMF, RESP:%d, USER=%s", res.StatusCode(), supi)

	if res.StatusCode() >= 300 {

		loggers.ErrorLogger().Minor("Recv Namf-COMM(N1N2-Transfer, MT) Resp(NACK) Result Code : %d(%s), USER:%s, RespToIF : %d",
			res.StatusCode(), res.ResponseString(), supi, http.StatusServiceUnavailable)

		callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusServiceUnavailable, supi)
		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)

		return s.RespondServiceUnavailable(ctx, "Service Unavailable")
	} else {
		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)
		loggers.InfoLogger().Comment("Recv Namf-COMM(N1N2-Transfer, MT) Resp(ACK), USER:%s, Resp:%d", supi, res.StatusCode())
	}

	loggers.InfoLogger().Comment("Succ MT(USER:%s), RespToIF(%d)", supi, http.StatusOK)
	callLogger.Info("SEND MT-SMS RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusOK, supi)

	ctx.String(http.StatusOK, "")
	return nil
}

func (s *MapServer) HandleMoResp(ctx echo.Context, supi string) (err error) {

	var n1n2 msg5g.Nfinterface
	//var mutex = &sync.Mutex{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	loggers.InfoLogger().Comment("Receive Mo Resp from MAP, user:%s", supi)
	callLogger.Info("RECV MO-RESP FROM SMSF-IF, USER=%s", supi)
	s.StatHttpif()

	ReqData := new(tcpmgr.HttpIfMoMsg)

	if err = ctx.Bind(ReqData); err != nil {
		loggers.ErrorLogger().Major("Invalid Internal Error : No-Body, supi(%s), respCode(%d)", supi, http.StatusBadRequest)
		callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

	amfSupi := "amf-" + supi

	rval, rdata := s.redisDao.GetSubBySUPI(amfSupi)
	if rval == -1 {
		rval, rdata = s.mysqlDao.GetSubInfoByKEY(amfSupi)
		if rval == -1 {
			loggers.InfoLogger().Comment("Invalid Internal Error : not find Subscriber, supi(%s), respCode(%d)", supi, http.StatusNotFound)
			callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusNotFound, supi)
			s.StatHttpRespCode(http.StatusNotFound, UplinkStat)
			return s.RespondNotFound(ctx)
		}
		s.redisDao.InsSub(amfSupi, rdata)
	}

	//mutex.Lock()
	smsContext := msg5g.UeSmsContextData{}
	err = json.Unmarshal(rdata, &smsContext)
	//mutex.Unlock()

	n1n2URL := fmt.Sprintf("/ue-contexts/%s/n1-n2-messages", supi)

	cpData := MakeCPData(ReqData.RpData)
	loggers.InfoLogger().Comment("cp-data : %x", cpData)

	contentsType := fmt.Sprintf("multipart/related;boundary=%s", common.SMSF_BOUNDARY)

	n1n2 = msg5g.N1N2_RESP{cpData, false, common.SMSF_BOUNDARY}
	n1n2MsgBody, err := n1n2.Make()

	cli, err := s.scpClient.OpenToNf(
		context.Background(),
		"AMF",
		smsContext.AmfId,
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
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer, MO RESP) because of NRF interworking, err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL N1N2-TRANSFER(CP-DATA, MO-RESP)(NRF) TO AMF, ERR:%s, USER=%s", err, supi)
		callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	}

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

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Namf-COMM(N1N2-Transfer, MO RESP) err : %s, USER:%s, RespToIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND FAIL N1N2-TRANSFER(CP-DATA, MO-RESP) TO AMF, ERR:%s, USER=%s", err, supi)
		callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		s.stats.IncErrCounter(InvalidN1N2SendHttpMsg)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))
	} else {
		loggers.InfoLogger().Comment("Send Succ Namf-COMM(N1N2-Transfer, MO RESP), USER:%s", supi)
		callLogger.Info("SEND SUCC N1N2-TRANSFER(CP-DATA, MO-RESP) TO AMF, USER=%s", supi)
	}

	s.stats.IncN1n2Counter(N1n2Total)

	if res.StatusCode() >= 300 {

		loggers.ErrorLogger().Minor("Recv Namf-EUCM(N1N2-Transfer, MO RESP) Resp(NACK) Result Code : %d(%s), USER:%s, RespToIF : %d",
			res.StatusCode(), res.ResponseString(), supi, http.StatusServiceUnavailable)

		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)
		loggers.InfoLogger().Comment("(map-n1n2)N1N2(MO RESP) Response err() : %s, USER:%s, ResptoIF : %d", err, supi, http.StatusBadRequest)
		callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusBadRequest, supi)
		return s.RespondSystemError(ctx, errcode.BadRequest(ctx.Request().URL.Path))

	} else {
		s.StatHttpRespCode(res.StatusCode(), N1n2Stat)
		loggers.InfoLogger().Comment("Recv Namf-EUCM(N1N2-Transfer, MO RESP) Resp(ACK), USER:%s, Resp:%d", supi, res.StatusCode())
	}
	callLogger.Info("RECV N1N2-TRANSFER(CP-DATA, MO-RESP) RESP FROM AMF, RESP:%d, USER=%s", res.StatusCode(), supi)

	loggers.InfoLogger().Comment("Succ MO-RESP(supi : %d), RespToIF(%d)", supi, http.StatusOK)
	callLogger.Info("SEND MO-RESP RESP TO SMSF-IF, RESP:%d, USER=%s", http.StatusOK, supi)
	ctx.String(http.StatusOK, "")
	return nil
}

func DecodeMultiPart(recvBody []byte, params map[string]string, supi string) (rpdata []byte, mms bool, ret int) {

	//var mutex = &sync.Mutex{}
	var bodyOfBinaryPart []byte
	var jsonContentsId string
	var binaryContentsId string
	var mmsFlag bool = false
	smsMsg := msg5g.SmsRequest{}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	bytesBuf := bytes.NewReader(recvBody)
	r := related.NewReader(bytesBuf, params)

	part, err := r.NextPart()
	for part != nil && err == nil {
		if part.Root {
			if common.IsContain(part.Header["Content-Type"], "application/json") {
				bodyOfJSONPart, err := ioutil.ReadAll(part)
				if err != nil {
					loggers.ErrorLogger().Major("Read Error JSON Data")
					return bodyOfBinaryPart, mmsFlag, SMSF_ERR
				}
				loggers.InfoLogger().Comment("JSON contents : %s", bodyOfJSONPart)

				//mutex.Lock()
				err = json.Unmarshal(bodyOfJSONPart, &smsMsg)
				//mutex.Unlock()
				if err != nil {
					loggers.ErrorLogger().Major("JSON unmarshalling Error, err:%s", err)
					return bodyOfBinaryPart, mmsFlag, SMSF_ERR
				}

				jsonContentsId = smsMsg.ContentsId
				mmsFlag = smsMsg.MmsFlag

				loggers.InfoLogger().Comment("ContentsId : %s, mmsFlag : %t", jsonContentsId, mmsFlag)
			} else {
				ulog.Error("Fail to parse Content-Type of the root part : %s, it should be application/json, USER:%s", part.Header["Content-Type"], supi)
				return bodyOfBinaryPart, mmsFlag, SMSF_ERR
			}
		} else {
			if common.IsContain(part.Header["Content-Type"], "application/vnd.3gpp.sms") {
				binaryContentsId = part.Header["Content-Id"][0][1 : len(part.Header["Content-Id"][0])-1]
				if len(binaryContentsId) == 0 {
					ulog.Error("Does not exist Content-Id, USER:%s", supi)
					return bodyOfBinaryPart, mmsFlag, SMSF_ERR
				} else if binaryContentsId != jsonContentsId {
					ulog.Error("binary data header contentId : %s, JSON data contentId : %s", binaryContentsId, jsonContentsId)
					return bodyOfBinaryPart, mmsFlag, SMSF_ERR
				}

				bodyOfBinaryPart, err = ioutil.ReadAll(part) //binary

				if err != nil {
					ulog.Error("Fail to read body of binary part : %s", err)
					return bodyOfBinaryPart, mmsFlag, SMSF_ERR
				}
				ulog.Info("Content-ID : %s", binaryContentsId)
				ulog.Info("Contents : %x", bodyOfBinaryPart)
			} else {
				ulog.Error("Fail to parse Content-Type of the part : %s, it should be application/3gpp.vnd.com", part.Header["Content-Type"])
				return bodyOfBinaryPart, mmsFlag, SMSF_ERR
			}

		}
		part, err = r.NextPart()
	}

	return bodyOfBinaryPart, mmsFlag, SMSF_OK
}
