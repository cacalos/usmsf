package common

/*
import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"golang.org/x/net/http2"
	"uangel.com/usmsf/interfaces"
)

// HTTPCliConf HTTP Client Configuration
type HTTPCliConf struct {
	InsecureSkipVerify         bool          //Secure Verify를 하지 않도록 설정
	IsWildcardHost             bool          // Host이름으로 인증서 사용 시 Wildcard을 사용하도록 설정
	DialTimeout                time.Duration // Client Connection Timeout
	DialKeepAlive              time.Duration // Client Connection Keep-Alive Timeout
	IdleConnTimeout            time.Duration // Client Connection Idle Timeout
	TLSConfig                  *tls.Config   // Client Connection TLS Configuration
	MaxHeaderListSize          uint32
	StrictMaxConcurrentStreams bool
}

// HTTPClient HTTP 연동용 Client
type HTTPClient struct {
	Scheme       string
	Host         string
	Address      string
	RootPath     string
	ProtoMajor   int
	IsTLS        bool
	TLSConfig    *tls.Config
	dialer       *net.Dialer
	transport    *http.Transport
	h2cTransport *http2.Transport
	client       *http.Client
	usedTime     int64 // 사용된 시간 unix nano time 값
	traceMgr     interfaces.TraceMgr
}

////////////////////////////////////////////////////////////////////////////////
// HTTPClient functions
////////////////////////////////////////////////////////////////////////////////

func NewNetHttpClient(cfg uconf.Config, keystore uregi.Cert) uclient.HTTP {
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{*keystore.GetCert()},
		RootCAs:            keystore.GetRootCA(),
		InsecureSkipVerify: true,
		// ALPN protocol 지정하는 부분입니다. http2 를 먼저 사용하고 , http2 를 지원하지 않으면 http/1.1 을 사용합니다.
		NextProtos: []string{"h2", "http/1.1"},
	}

	// http1 transport 를 생성할 때  tls config 가 있으면 , http2  transport 를 지원하지 않습니다.
	transport := http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 8192, //  이 값을 설정해 주지 않으면 timewait 이 계속 발생하게 됩니다.
	}

	// 따라서 , 별도로 http2 transport 설정을 이렇게 호출 해 주어야 합니다.
	http2.ConfigureTransport(&transport)
	return &NetHttpClient{&transport}

}

func (c *HTTPClient) Reconnect() error {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	if c.ProtoMajor == 2 && !c.IsTLS {
		transport := &http.Transport{
			MaxIdleConns:        8192,
			MaxIdleConnsPerHost: 8192,
			MaxConnsPerHost:     8192,
			IdleConnTimeout:     60 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if c.Address != "" {
					addr = c.Address
				}
				return dialer.DialContext(ctx, network, addr)
			},
		}

		http2.ConfigureTransport(transport)
		c.client = &http.Client{
			Transport: transport,
		}

		return nil
	}

	transport := &http.Transport{
		MaxIdleConns:        8192,
		MaxIdleConnsPerHost: 8192,
		MaxConnsPerHost:     8192,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     c.TLSConfig,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if c.Address != "" {
				addr = c.Address
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	if c.ProtoMajor >= 2 {
		http2.ConfigureTransport(transport)
	}
	c.client = &http.Client{
		Transport: transport,
	}

	return nil
}

// Close 전달된 SEPP Client를 Close 한다.
func (c *HTTPClient) Close() {
	if c.h2cTransport != nil {
		c.h2cTransport.CloseIdleConnections()
	}
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
}

// Call 전달된 정보를 바탕으로 요청을 만들어 설정된 서버와 통신하고 결과를 받는다.
func (c *HTTPClient) Call(method, path string, header http.Header, body []byte) (*http.Response, []byte, error) {
	url := c.RootPath + path

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.With(ulog.Fields{"error": err.Error()}).Error("Failed to make tls request")
		return nil, nil, err
	}

	if header != nil {
		for k := range header {
			req.Header.Set(k, header.Get(k))
		}
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		logger.With(ulog.Fields{"error": err.Error()}).Error("Failed to call HTTP message")
		return nil, nil, err
	}
	defer rsp.Body.Close()

	rspbody, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		logger.With(ulog.Fields{"error": err.Error()}).Error("Failed to read HTTP response")
		return nil, nil, err
	}

	return rsp, rspbody, nil
}

// Forward 전달된 Request를 설전된 Server로 Forward 한다.
func (c *HTTPClient) Forward(req *http.Request) (*http.Response, error) {
	if c.h2cTransport != nil {
		return c.h2cTransport.RoundTrip(req)
	}
	return c.transport.RoundTrip(req)
}

// dialHTTP HTTP 1.x, TLS 용 Connection 생성 함수
func (c *HTTPClient) dialHTTP(ctx context.Context, network, addr string) (net.Conn, error) {
	if c.Address != "" {
		return c.dialer.DialContext(ctx, network, c.Address)
	}
	return c.dialer.DialContext(ctx, network, addr)
}

// dialH2C H2C 용 Connection 생성 함수
func (c *HTTPClient) dialH2C(network, addr string) (net.Conn, error) {
	if c.Address != "" {
		return net.Dial(network, c.Address)
	}
	return net.Dial(network, addr)
}
*/
