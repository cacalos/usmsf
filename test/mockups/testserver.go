package mockups

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"camel.uangel.com/ua5g/ubsf.git/implements/utils"
	"camel.uangel.com/ua5g/ubsf.git/msg5g"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// TestServer Test용 웹 서버 (from SEPP)
type TestServer struct {
	Addr        string
	Host        string
	IsTLS       bool
	IsHTTP2     bool
	Engine      *gin.Engine
	HandleFunc  gin.HandlerFunc
	TLSConfig   *tls.Config
	HTTP2SvrCfg *http2.Server
	Listener    *http.Server
}

// NewTestServer TLS Server를 생성한다.
func NewTestServer(addr, host string, isTLS, isHTTP2 bool, handle gin.HandlerFunc) (*TestServer, error) {
	if os.Getenv("UBSF_HOME") == "" {
		os.Setenv("UBSF_HOME", utils.GetModuleRootPath("/mockups"))
	}

	s := &TestServer{
		Addr:       addr,
		Host:       host,
		IsTLS:      isTLS,
		IsHTTP2:    isHTTP2,
		HandleFunc: handle,
	}

	// k8s 에 배포하는 경우, 다음의 코드는 수정 되어야 할 것.
	// 필요한 경로는 설정 파일로부터 가져와야 함.
	peercert := os.Getenv("UBSF_HOME") + "/resources/certs/rootca.com/rootca.crt"
	localcert := os.Getenv("UBSF_HOME") + "/resources/certs/" + s.Host + "/server.crt"
	localkey := os.Getenv("UBSF_HOME") + "/resources/certs/" + s.Host + "/server.key"

	if isTLS {
		caCert, err := ioutil.ReadFile(filepath.FromSlash(peercert))
		if err != nil {
			logger.Fatal(err.Error())
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		cert, err := tls.LoadX509KeyPair(filepath.FromSlash(localcert), filepath.FromSlash(localkey))
		if err != nil {
			logger.Fatal(err.Error())
			return nil, err
		}
		s.TLSConfig = &tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
			Certificates: []tls.Certificate{cert},
		}
	} else {
		s.TLSConfig = nil
	}

	s.Engine = gin.New()
	var handler http.Handler
	handler = s.Engine
	if isHTTP2 {
		s.HTTP2SvrCfg = &http2.Server{}
		if !isTLS {
			handler = h2c.NewHandler(s.Engine, &http2.Server{})
		}
	}

	// HTTP Server
	s.Listener = &http.Server{
		Addr:      s.Addr,
		Handler:   handler,
		TLSConfig: s.TLSConfig,
	}

	if isHTTP2 {
		//http2.VerboseLogs = true
		http2.ConfigureServer(s.Listener, s.HTTP2SvrCfg)
	}

	s.Engine.Any("/*path", s.Handle)

	return s, nil
}

// NewTestServerWithHTTP2TLS TLS 설정을 사용하는 버전의 생성자
func NewTestServerWithHTTP2TLS(addr, host string, tlsConfig *tls.Config, handle gin.HandlerFunc) (*TestServer, error) {
	s := &TestServer{
		Addr:       addr,
		Host:       host,
		IsTLS:      true,
		IsHTTP2:    true,
		HandleFunc: handle,
	}

	s.TLSConfig = tlsConfig
	s.Engine = gin.New()

	s.HTTP2SvrCfg = &http2.Server{}
	s.Listener = &http.Server{
		Addr:      s.Addr,
		Handler:   s.Engine,
		TLSConfig: s.TLSConfig,
	}
	http2.ConfigureServer(s.Listener, s.HTTP2SvrCfg)

	s.Engine.Any("/*path", s.Handle)

	return s, nil
}

//ListenAndServe 해당 Server를 Listen 및 서비스 한다.
func (s *TestServer) ListenAndServe() error {
	if s.IsTLS {
		return s.Listener.ListenAndServeTLS("", "")
	}
	return s.Listener.ListenAndServe()
}

// Start Server를 구동한다.
func (s *TestServer) Start() {
	waitchnl := make(chan string)
	go func() {
		waitchnl <- "Test-Server start " + s.Addr
		err := s.ListenAndServe()
		if err != nil {
			//logger.Info("Shutting down the Test-Server %v", s.Addr)
		} else {
			logger.With(ulog.Fields{"error": err.Error()}).Fatal("Failed to listen and serve Test-Server %v", s.Addr)
		}
	}()
	<-waitchnl
	//logger.Info(<-waitchnl)
}

// Stop Test Server의 구동을 멈춘다.
func (s *TestServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Listener.Shutdown(ctx); err != nil {
		logger.With(ulog.Fields{"error": err.Error()}).Fatal("Failed to shutdown Test-Server %v", s.Addr)
	}
}

// Handle HTTP message Handle
func (s *TestServer) Handle(ctx *gin.Context) {
	if s.HandleFunc == nil {
		s.ResponseError(ctx, 404, "Not Found", "'%s/%s' is not found.", ctx.Request.Host, ctx.Request.URL.String())
		return
	}
	s.HandleFunc(ctx)
}

// ResponseError 전달된 에러를 반환한다.
func (s *TestServer) ResponseError(ctx *gin.Context, code int, title string, formatstr string, args ...interface{}) {
	pd := &msg5g.ProblemDetails{
		Status: 404,
		Title:  "Not Found",
		Detail: fmt.Sprintf("'%s/%s' is not found.", ctx.Request.Host, ctx.Request.URL.String()),
	}
	rbody, err := json.Marshal(pd)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
	} else {
		ctx.Data(pd.Status, "application/problem+json", rbody)
	}
}
