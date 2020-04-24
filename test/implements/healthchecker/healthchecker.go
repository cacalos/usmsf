package healthchecker

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"github.com/heptiolabs/healthcheck"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/configmgr"
)

var loggers = common.SamsungLoggers()

// HealthChecker : healthchecker용 구조체 정의
type HealthChecker struct {
	httpServer           *http.Server
	uconfMgr             uconf.Config
	svcconfMgr           *configmgr.ConfigServer
	conf                 healthCheckerConfig
	handlingFuncsByProbe map[string]functionsByProbe
	handler              healthcheck.Handler

	// readiness check가 성공한 경우 true로 설정한다.
	// (매번 readiness check를 위해 다른 pod와 통신을 하는 경우 성능 저하 우려가 있어 cache로 사용)
	isReady bool
}

// Probe별 필요 함수들을 모아 둔 구조체
type functionsByProbe struct {
	checkerAddingFunc func(name string, check healthcheck.Check)
	handlingFunc      func() error
}

type healthCheckerConfig struct {
	listenPort int

	// key of map: "liveness", "readiness", ...
	configsByProbe map[string]*configByProbe
}

// Probe별 config
type configByProbe struct {

	// true인 경우에만 해당 probe(e.g. liveness probe)에 대한 checker 동작
	enabled bool

	// true:  백그라운드에서 주기적으로 liveness 검사 (주기: asyncCheckInterval)
	// false: kubelet으로부터 liveness probe 수신 시에만 liveness 검사
	useAsyncCheck bool

	// useAsyncCheck=true인 경우에만 유효
	asyncCheckInterval time.Duration

	simMode simModeConfig
}

// 특정 상황을 시뮬레이션하여 시험할 때 사용하는 config 값들 정의
type simModeConfig struct {

	// 시험 편의를 위한 것으로, true인 경우 실제 정상 동작하지 않고 shouldReturnError 값에 따라 동작한다.
	// (실제 상용에서는 반드시 false로 설정되어야 함.)
	enabled bool

	// true인 경우, probe에 대한 응답으로 실패를, false인 경우 성공을 반환한다.
	shouldReturnError bool
}

// NewHealthChecker : healthcheck instance를 생성하여 반환한다.
func NewHealthChecker(
	uconfMgr uconf.Config,
	svcconfMgr *configmgr.ConfigServer,
) *HealthChecker {

	hc := &HealthChecker{
		httpServer: &http.Server{},
		uconfMgr:   uconfMgr,
		svcconfMgr: svcconfMgr,
		isReady:    false,
	}

	err := hc.Init()
	if err != nil {
		loggers.ErrorLogger().Major("Failed to initialize Health Checker")
		return nil
	}

	return hc
}

// Init : 구조체 instance를 초기화한다.
func (r *HealthChecker) Init() error {

	r.handler = healthcheck.NewHandler()

	r.initHandlingFuncMap()

	err := r.loadConfig()
	if err != nil {
		loggers.ErrorLogger().Major("Failed to load config")
		return err
	}

	r.httpServer.Addr = fmt.Sprintf(":%d", r.conf.listenPort)

	r.httpServer.Handler = r.initHandler()

	return nil
}

// Probe별 관련 함수들을 등록한다.
func (r *HealthChecker) initHandlingFuncMap() {

	// 아래 map에서 key인 "liveness"와 같은 probe 이름은,
	// config 파일의 health-checker object 내 probe object 이름과 일치해야 한다.
	// e.g. health-checker.liveness
	r.handlingFuncsByProbe = map[string]functionsByProbe{
		"liveness":  {r.handler.AddLivenessCheck, r.checkLiveness},
		"readiness": {r.handler.AddReadinessCheck, r.checkReadiness},
	}
}

func (r *HealthChecker) loadConfig() error {

	r.conf.configsByProbe = map[string]*configByProbe{}

	confRootName := "smsf.health-checker"

	confRoot := r.uconfMgr.GetConfig(confRootName)
	if confRoot == nil {
		err := fmt.Errorf("Config %#v not found", confRootName)
		loggers.ConfigLogger().Major(err.Error())
		return err
	}

	r.conf.listenPort = confRoot.GetInt("listen-port", 8080)

	for probeName := range r.handlingFuncsByProbe {

		uconfByProbe := confRoot.GetConfig(probeName)
		if uconfByProbe != nil {

			configByProbe := new(configByProbe)

			configByProbe.enabled = uconfByProbe.GetBoolean("enabled", false)
			configByProbe.useAsyncCheck = uconfByProbe.GetBoolean("use-async-check", false)
			configByProbe.asyncCheckInterval = uconfByProbe.GetDuration("async-check-interval", 5*time.Second)

			configTester := uconfByProbe.GetConfig("simulation-mode")
			if configTester != nil {
				configByProbe.simMode.enabled = configTester.GetBoolean("enabled", false)
				configByProbe.simMode.shouldReturnError = configTester.GetBoolean("return-error", false)
			}

			r.conf.configsByProbe[probeName] = configByProbe
		}
	}

	loggers.ConfigLogger().Comment("%#v", &r.conf)

	return nil
}

func (r *HealthChecker) livenessConfig() *configByProbe {
	return r.conf.configsByProbe["liveness"]
}

func (r *HealthChecker) readinessConfig() *configByProbe {
	return r.conf.configsByProbe["readiness"]
}

// handler를 초기화하여 반환한다.
func (r *HealthChecker) initHandler() healthcheck.Handler {

	for probeName, functions := range r.handlingFuncsByProbe {

		cfg := r.conf.configsByProbe[probeName]
		if cfg == nil {
			loggers.ErrorLogger().Comment("config for %#v not found", probeName)
			continue
		}

		if !cfg.enabled {
			loggers.InfoLogger().Comment("Set to not check %v", probeName)
			continue
		}

		checkerAddingFunc := functions.checkerAddingFunc
		var checkingFunc healthcheck.Check

		if cfg.useAsyncCheck {
			checkingFunc = healthcheck.Async(functions.handlingFunc, cfg.asyncCheckInterval)
		} else {
			checkingFunc = functions.handlingFunc
		}

		if probeName != "" && checkerAddingFunc != nil && checkingFunc != nil {
			checkerAddingFunc(probeName, checkingFunc)
			loggers.InfoLogger().Comment("checkerAddingFunc: %#v, checkingFunc: %#v", checkerAddingFunc, checkingFunc)
		}
	}

	return r.handler
}

// Liveness probe 처리 함수
func (r *HealthChecker) checkLiveness() error {

	cfg := r.livenessConfig()

	if cfg.simMode.enabled && cfg.simMode.shouldReturnError {
		return errors.New("[checkLiveness] Returning error (by simulation mode)")
	}

	loggers.InfoLogger().Comment("[checkLiveness] Alive")

	return nil
}

// Readiness probe 처리 함수
func (r *HealthChecker) checkReadiness() error {

	cfg := r.readinessConfig()

	if cfg.simMode.enabled {

		var err error = nil

		if cfg.simMode.shouldReturnError {
			err = errors.New("[checkReadiness] Not ready to receive traffic (by simulation mode)")
			loggers.InfoLogger().Major(err.Error())
		} else {
			loggers.InfoLogger().Comment("[checkReadiness] Ready to receive traffic (by simulation mode)")
			err = nil
		}

		return err
	}

	if !r.isReady {
		// UCCMS로 HTTP GET 전송 후, 받은 응답 메시지의 status code가 "실패"이면 error를, 그렇지 않으면 nil 반환
		err := r.svcconfMgr.GetDecisionConfig()
		if err == nil {
			loggers.InfoLogger().Comment("Result from UCCMS is OK; Now ready to receive traffic")
			r.isReady = true
		}
	}

	if !r.isReady {
		return errors.New("Not ready to receive traffic yet")
	}

	loggers.InfoLogger().Comment("[checkReadiness] Ready to receive traffic")

	return nil
}

// Start : health checker를 시작한다.
func (r *HealthChecker) Start() {

	waitchnl := make(chan string)

	exec.SafeGo(func() {
		waitchnl <- "Health Check Server started on http://" + r.httpServer.Addr
		err := r.httpServer.ListenAndServe()
		if err != nil {
			loggers.ErrorLogger().Critical("Failed to listen and serve Health Checker Server on http://%v: %v", r.httpServer.Addr, err.Error())
			os.Exit(1)
		}
	})

	loggers.EventLogger().Data(<-waitchnl)
}
