package common

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
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
	"golang.org/x/net/http2"
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

// NewHTTPClient HTTPClient를 생성해 반환한다.
func NewHTTPClient(cliconf *HTTPCliConf, scheme, host, addr string, protoMajor int, traceMgr interfaces.TraceMgr) (*HTTPClient, error) {
	c := &HTTPClient{
		Scheme:     scheme,
		Host:       host,
		Address:    addr,
		ProtoMajor: protoMajor,
		traceMgr:   traceMgr,
	}
	if scheme == "https" && cliconf.TLSConfig == nil {
		return nil, errcode.SystemError("No TLS Certerfication information, host:%s, addr:%s, tlsConfig is null? : %t",
			host, addr, cliconf.TLSConfig == nil)
	}
	if addr == "" {
		c.RootPath = scheme + "://" + host
	} else {
		c.RootPath = scheme + "://" + addr
	}

	if scheme != "https" && protoMajor >= 2 {

		c.transport = &http.Transport{
			MaxIdleConns:        8192,
			MaxIdleConnsPerHost: 8192,
			MaxConnsPerHost:     8192,
			IdleConnTimeout:     cliconf.IdleConnTimeout,
			DialTLS:             c.dialH2C,
		}

		http2.ConfigureTransport(c.transport)
		c.client = &http.Client{
			Transport: c.transport,
		}

		return c, nil
	}

	c.dialer = &net.Dialer{
		Timeout:   cliconf.DialTimeout,
		KeepAlive: cliconf.DialKeepAlive,
		DualStack: true,
	}

	if cliconf.TLSConfig != nil {
		c.IsTLS = true
		c.TLSConfig = cliconf.TLSConfig.Clone()
		if !cliconf.InsecureSkipVerify {
			c.TLSConfig.InsecureSkipVerify = false
			if len(addr) > 0 && addr[0] != '[' && addr[0] < '0' && addr[0] > '9' {
				idx := strings.LastIndex(addr, ":")
				c.TLSConfig.ServerName = addr[:idx]
			} else {
				if cliconf.IsWildcardHost {
					c.TLSConfig.ServerName = "*." + host
				} else {
					c.TLSConfig.ServerName = host
				}
			}
		}
	} else {
		c.IsTLS = false
	}

	c.transport = &http.Transport{
		MaxIdleConns:        8192,
		MaxIdleConnsPerHost: 8192,
		MaxConnsPerHost:     8192,
		IdleConnTimeout:     cliconf.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     c.TLSConfig,
		DialContext:         c.dialHTTP,
	}

	if protoMajor >= 2 {
		http2.ConfigureTransport(c.transport)
	}
	c.client = &http.Client{
		Transport: c.transport,
	}

	return c, nil
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
		loggers.ErrorLogger().Major("Failed to make TLS request: error=% #v", err.Error())
		return nil, nil, err
	}

	if header != nil {
		for k := range header {
			req.Header.Set(k, header.Get(k))
		}
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		loggers.ErrorLogger().Major("Failed to send HTTP message: error=% #v", err.Error())
		return nil, nil, err
	}
	defer rsp.Body.Close()

	rspbody, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		loggers.ErrorLogger().Major("Failed to read HTTP response: error=% #v", err.Error())
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
