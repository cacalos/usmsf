package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ustat"
	metrics "github.com/rcrowley/go-metrics"
	"golang.org/x/net/http2"
)

// HTTPMetrics HTTP Metrics
type HTTPMetrics struct {
	mutex        sync.Mutex
	reqs         uint64 // 1XX를 제외한 final response
	rsps         uint64
	rsps1xx      uint64
	rsps2xx      uint64
	rsps3xx      uint64
	rsps4xx      uint64
	rsps5xx      uint64
	rspsunk      uint64
	retryCount   uint64
	retrySuccess uint64
	retryFail    uint64
	errors       map[int]uint64
	success      ustat.Timer
	rsptime      ustat.Timer
	started      bool
	first        time.Time
	last         time.Time
	activate     ustat.Timer
	deactivate   ustat.Timer
	uplink       ustat.Timer
	sdmNoti      ustat.Timer

	nfRegister   ustat.Timer
	nfHeartBeat  ustat.Timer
	nfDeregister ustat.Timer
	nfPatch      ustat.Timer
	subs         ustat.Timer
	subsPatch    ustat.Timer
	unsubs       ustat.Timer
	receiveNoti  ustat.Timer
	nfList       ustat.Timer
	nfGet        ustat.Timer

	nfCounter     ustat.Counter
	discoverHisto map[string]ustat.Histogram
}

// HTTPClient HTTP Client
type HTTPClient struct {
	TargetIP            string
	TargetHost          string
	LocalHost           string
	CaCertsFile         string
	CertFile            string
	PrivateKeyFile      string
	IsTLS               bool
	IsHTTP2             bool
	ConnTimeout         time.Duration
	ConnKeepAlive       time.Duration
	ConnDualStack       bool
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	RootPath            string
	TLSConfig           *tls.Config
	Client              *http.Client
}

////////////////////////////////////////////////////////////////////////////////
// Functions of HTTPMetrics
////////////////////////////////////////////////////////////////////////////////

// NewHTTPMetrics HTTP 통계
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		success:    ustat.GetOrRegisterTimer("success", nil),
		rsptime:    ustat.GetOrRegisterTimer("rsptime", nil),
		activate:   ustat.GetOrRegisterTimer("activate", nil),
		deactivate: ustat.GetOrRegisterTimer("deactivate", nil),
		uplink:     ustat.GetOrRegisterTimer("uplink", nil),
		sdmNoti:    ustat.GetOrRegisterTimer("sdmNoti", nil),

		nfRegister:   ustat.GetOrRegisterTimer("nfRegister", nil),
		nfHeartBeat:  ustat.GetOrRegisterTimer("nfHeartBeat", nil),
		nfDeregister: ustat.GetOrRegisterTimer("nfDeregister", nil),
		nfPatch:      ustat.GetOrRegisterTimer("nfPatch", nil),
		subs:         ustat.GetOrRegisterTimer("Subs", nil),
		subsPatch:    ustat.GetOrRegisterTimer("subsPatch", nil),
		unsubs:       ustat.GetOrRegisterTimer("Unsubs", nil),
		receiveNoti:  ustat.GetOrRegisterTimer("Noti", nil),
		nfList:       ustat.GetOrRegisterTimer("nfList", nil),
		nfGet:        ustat.GetOrRegisterTimer("nfGet", nil),

		nfCounter:     ustat.GetOrRegisterCounter("nfCounter", nil),
		discoverHisto: map[string]ustat.Histogram{},
		errors:        make(map[int]uint64),
	}
}
func (m *HTTPMetrics) initHistogram(name string) {
	m.discoverHisto[name] = ustat.GetOrRegisterHistogram(name, nil, metrics.NewExpDecaySample(1028, 0.015))
}

// Start HTTP 통계 수집을 시작한다.
func (m *HTTPMetrics) Start() time.Time {
	now := time.Now()
	atomic.AddUint64(&m.reqs, 1)
	if !m.started {
		m.first = now
		m.started = true
	}
	return now
}

// Stop HTTP 통계 수집을 종료한다.
func (m *HTTPMetrics) Stop(start time.Time, code int) {
	now := time.Now()
	m.last = now
	if code < 200 {
		atomic.AddUint64(&m.rsps1xx, 1)
		return
	}

	atomic.AddUint64(&m.rsps, 1)
	m.rsptime.Update(now.Sub(start))

	if code < 300 {
		atomic.AddUint64(&m.rsps2xx, 1)
		m.success.Update(now.Sub(start))

	} else {
		if code < 400 {
			atomic.AddUint64(&m.rsps3xx, 1)
		} else if code < 500 {
			atomic.AddUint64(&m.rsps4xx, 1)
		} else if code < 600 {
			atomic.AddUint64(&m.rsps5xx, 1)
		} else {
			atomic.AddUint64(&m.rspsunk, 1)
		}
		m.mutex.Lock()
		defer m.mutex.Unlock()
		v, ok := m.errors[code]
		if ok {
			m.errors[code] = v + 1
		} else {
			m.errors[code] = 1
		}
	}
}

func (m *HTTPMetrics) ActivateStop(start time.Time, code int) {
	now := time.Now()
	m.activate.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) DeactivateStop(start time.Time, code int) {
	now := time.Now()
	m.deactivate.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) UplinkStop(start time.Time, code int) {
	now := time.Now()
	m.uplink.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) SdmNotiStop(start time.Time, code int) {
	now := time.Now()
	m.sdmNoti.Update(now.Sub(start))
	m.Stop(start, code)
}

func PrintServiceStat(servicename string, timer ustat.Timer, duration uint64) {
	snap := timer.Snapshot()
	if snap.Count() > 0 {
		fmt.Printf("[%-20s] : %v, TPS=%v, (1m tps = %.2f)\n", servicename, snap.Count(), uint64(snap.Count())/duration, snap.Rate1())
	}

}

// Report HTTP 통계를 출력한다.
func (m *HTTPMetrics) Report(prResult, prTime, prError bool) {
	duration := uint64(1)
	if m.started {
		duration = uint64(m.last.Sub(m.first) / time.Second)
		if duration == 0 {
			duration = 1
		}
	}

	fmt.Printf("[Request ] : %v , ( In Progress : %d )\n", m.reqs, m.reqs-m.rsps)
	fmt.Printf("[Response] : %v, TPS=%v, (1m tps = %.2f)\n", m.rsps, m.rsps/duration, m.rsptime.Rate1())
	fmt.Printf("[Success ] : %v, TPS=%v, (1m tps = %.2f) , RATE=%.2f\n", m.rsps2xx, m.rsps2xx/duration, m.success.Rate1(), float64(m.rsps2xx)/float64(m.rsps)*100)

	PrintServiceStat("Activate", m.activate, duration)
	PrintServiceStat("Deactivate", m.deactivate, duration)
	PrintServiceStat("Uplink", m.uplink, duration)
	PrintServiceStat("SdmNoti", m.sdmNoti, duration)

	fmt.Printf("[NF Count] : %d\n", m.nfCounter.Count())

	for tpe, histo := range m.discoverHisto {
		fmt.Printf("Avg Disc for %s : %.2f\n", tpe, histo.Mean())
	}

	if prResult {
		fmt.Printf("[StatusCodes] : \n")
		fmt.Printf("  1XX = %v, 2XX = %v, 3XX = %v, 4XX = %v, 5xx = %v, oth = %v , retry = %v , retry-success = %v , retry-fail = %v\n",
			m.rsps1xx, m.rsps2xx, m.rsps3xx, m.rsps4xx, m.rsps5xx, m.rspsunk, m.retryCount, m.retrySuccess, m.retryFail)
	}
	if prTime {

		fmt.Printf("[Rsp Time] : \n")
		fmt.Printf("  avg = %.2f msec, min= %.2f msec, max = %.2f msec, 99%% = %.2f msec\n",
			m.rsptime.Mean()/float64(time.Millisecond),
			float64(m.rsptime.Min())/float64(time.Millisecond),
			float64(m.rsptime.Max())/float64(time.Millisecond),
			float64(m.rsptime.Percentile(0.99))/float64(time.Millisecond),
		)
	}
	if prError && len(m.errors) > 0 {
		fmt.Printf("[Error   ] :\n")
		for c, v := range m.errors {
			fmt.Printf("  %d = %v\n", c, v)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Functions of HTTPClient
////////////////////////////////////////////////////////////////////////////////

// NewHTTPClientWithConfig Test HTTP Client를 Configuration 기반으로 생성해 반환한다.
func NewHTTPClientWithConfig(cfg uconf.Config, name string) (*HTTPClient, error) {
	hcfg := cfg.GetConfig(name)
	if hcfg == nil {
		hcfg = cfg
	}
	targetHost := hcfg.GetString("target-host", cfg.GetString("target-host", ""))
	if targetHost == "" {
		fmt.Printf("Not found configuration data. (%v.target-host or target-host is emtpy)", name)
		return nil, fmt.Errorf("Not found configuration data. (%v.target-host or target-host is emtpy)", name)
	}
	targetIP := hcfg.GetString("target-address", cfg.GetString("target-address", ""))
	localHost := hcfg.GetString("local-host", cfg.GetString("local-host", ""))
	cacerts := hcfg.GetString("cacerts-file", cfg.GetString("cacerts-file", ""))
	cert := hcfg.GetString("cert-file", cfg.GetString("cert-file", ""))
	key := hcfg.GetString("private-key-file", cfg.GetString("private-key-file", ""))
	isTLS := hcfg.GetBoolean("tls", cfg.GetBoolean("tls", true))
	isHTTP2 := hcfg.GetBoolean("http2", cfg.GetBoolean("http2", true))
	connTimeout := hcfg.GetDuration("conn-timeout", cfg.GetDuration("conn-timeout", 30*time.Second))
	connKeepAlive := hcfg.GetDuration("conn-keep-alive", cfg.GetDuration("conn-keep-alive", 30*time.Second))
	connDualStack := hcfg.GetBoolean("conn-dual-stack", cfg.GetBoolean("conn-dual-stack", true))
	maxIdleConns := hcfg.GetInt("max-idle-conns", cfg.GetInt("max-idle-conns", 8192))
	maxIdleConnsPerHost := hcfg.GetInt("max-idle-conns-per-host", cfg.GetInt("max-idle-conns-per-host", 8192))
	maxConnsPerHost := hcfg.GetInt("max-conns-per-host", cfg.GetInt("max-conns-per-host", 8192))

	c := &HTTPClient{
		TargetIP:            targetIP,
		TargetHost:          targetHost,
		LocalHost:           localHost,
		CaCertsFile:         cacerts,
		CertFile:            cert,
		PrivateKeyFile:      key,
		IsTLS:               isTLS,
		IsHTTP2:             isHTTP2,
		ConnTimeout:         connTimeout,
		ConnKeepAlive:       connKeepAlive,
		ConnDualStack:       connDualStack,
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		MaxConnsPerHost:     maxConnsPerHost,
	}
	err := c.init()
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewHTTPClient Test HTTP Client를 생성해 반환한다.
func NewHTTPClient(targetIP, targetHost, localHost, cacerts, cert, key string, isTLS, isHTTP2 bool) (*HTTPClient, error) {
	c := &HTTPClient{
		TargetIP:            targetIP,
		TargetHost:          targetHost,
		LocalHost:           localHost,
		CaCertsFile:         cacerts,
		CertFile:            cert,
		PrivateKeyFile:      key,
		IsTLS:               isTLS,
		IsHTTP2:             isHTTP2,
		ConnTimeout:         30 * time.Second,
		ConnKeepAlive:       30 * time.Second,
		ConnDualStack:       true,
		MaxIdleConns:        8192,
		MaxIdleConnsPerHost: 8192,
		MaxConnsPerHost:     8192,
	}

	err := c.init()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *HTTPClient) reconnect() error {
	if c.IsHTTP2 && !c.IsTLS {
		transport := &http2.Transport{
			AllowHTTP:                  true,
			TLSClientConfig:            c.TLSConfig,
			StrictMaxConcurrentStreams: false,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				if c.TargetIP != "" {
					addr = c.TargetIP
				}
				if c.IsTLS {
					return tls.Dial(network, addr, cfg)
				}
				return net.Dial(network, addr)
			},
		}
		c.Client = &http.Client{
			Transport: transport,
		}
		c.Client.Timeout = 30 * time.Second

		return nil
	}
	dialer := &net.Dialer{
		Timeout:   c.ConnTimeout,
		KeepAlive: c.ConnKeepAlive,
		DualStack: c.ConnDualStack,
	}
	transport := &http.Transport{
		TLSClientConfig: c.TLSConfig,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if c.TargetIP != "" {
				addr = c.TargetIP
			}
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		MaxConnsPerHost:     c.MaxConnsPerHost,
		IdleConnTimeout:     30 * time.Second,
	}
	if c.IsHTTP2 {
		http2.ConfigureTransport(transport)
	}
	c.Client = &http.Client{
		Transport: transport,
	}
	c.Client.Timeout = 30 * time.Second
	return nil
}

// NewHTTPClient Test HTTP Client를 생성해 반환한다.
func (c *HTTPClient) init() error {

	scheme := "http://"
	if c.IsTLS {
		caCert, err := ioutil.ReadFile(filepath.FromSlash(c.CaCertsFile))
		if err != nil {
			fmt.Printf("Failed to read Trusted-CA-Certifications. (cacerts=%v, err=%v)\n", c.CaCertsFile, err)
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		cert, err := tls.LoadX509KeyPair(filepath.FromSlash(c.CertFile), filepath.FromSlash(c.PrivateKeyFile))
		if err != nil {
			fmt.Printf("Failed to read Local Certification & Private key. (cert=%v, key=%v, err=%v)\n", c.CertFile, c.PrivateKeyFile, err)
			return err
		}
		c.TLSConfig = &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{cert},
			ServerName:   c.TargetHost,
		}
		scheme = "https://"
	}
	c.RootPath = scheme + c.TargetHost

	c.reconnect()
	return nil
}

// func (c *HTTPClient) SendPerf(method, path string, hdrs map[string]string, body []byte) (*http.Response, error) {
// 	return c.SendPerfVerbose(false, method, path, hdrs, body)
// }

// SendPerfVerbose 성능 테스트 용 Request 전송, Body를 읽지 않고 무사한다.
func (c *HTTPClient) SendPerfVerbose(metrics *HTTPMetrics, verbose bool, method, path string, hdrs map[string]string, body []byte) (*http.Response, error) {
	return c.sendPerfVerbose(metrics, verbose, method, path, hdrs, body, false)
}

func (c *HTTPClient) sendPerfVerbose(metrics *HTTPMetrics, verbose bool, method, path string, hdrs map[string]string, body []byte, retry bool) (*http.Response, error) {
	var rbuf io.Reader
	if body != nil {
		rbuf = bytes.NewBuffer(body)
	} else {
		rbuf = nil
	}
	req, err := http.NewRequest(method, c.RootPath+path, rbuf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Host", c.TargetHost)
	if hdrs != nil {
		for k, v := range hdrs {
			req.Header.Set(k, v)
		}
	}

	if verbose {
		fmt.Println("====================================>>>>>>")
		b, _ := httputil.DumpRequest(req, true)
		fmt.Println(string(b))
		fmt.Println("")
	}

	begin := time.Now()
	rsp, err := c.Client.Do(req)
	if err != nil {

		c.reconnect()

		if retry == false {
			end := time.Now()

			diff := end.Sub(begin)
			if timeout-diff > 0 {
				//time.Sleep(timeout - diff)
			}
			time.Sleep(timeout)
			atomic.AddUint64(&metrics.retryCount, 1)

			newcc, err := NewHTTPClient(c.TargetIP, c.TargetHost, c.LocalHost, c.CaCertsFile, c.CertFile, c.PrivateKeyFile, c.IsTLS, c.IsHTTP2)
			if err == nil {
				rsp, err = newcc.sendPerfVerbose(metrics, verbose, method, path, hdrs, body, true)

				if err != nil {
					atomic.AddUint64(&metrics.retryFail, 1)
				} else {
					if rsp.StatusCode < 300 {
						atomic.AddUint64(&metrics.retrySuccess, 1)

					} else {
						atomic.AddUint64(&metrics.retryFail, 1)
					}
				}
			}

			return rsp, err
		}
		return nil, err

	}

	if verbose {
		fmt.Println("<<<<<<====================================")
		b, _ := httputil.DumpResponse(rsp, true)
		fmt.Println(string(b))
		fmt.Println("")
	}

	if rsp != nil {
		defer rsp.Body.Close()
	}

	io.Copy(ioutil.Discard, rsp.Body)

	if rsp.StatusCode >= 500 && retry == false {
		end := time.Now()

		diff := end.Sub(begin)
		if timeout-diff > 0 {
			//time.Sleep(timeout - diff)
		}
		time.Sleep(timeout)

		atomic.AddUint64(&metrics.retryCount, 1)

		newcc, err := NewHTTPClient(c.TargetIP, c.TargetHost, c.LocalHost, c.CaCertsFile, c.CertFile, c.PrivateKeyFile, c.IsTLS, c.IsHTTP2)
		if err == nil {
			rsp, err = newcc.sendPerfVerbose(metrics, verbose, method, path, hdrs, body, true)
			if err != nil {
				atomic.AddUint64(&metrics.retryFail, 1)
			} else {
				if rsp.StatusCode < 300 {
					atomic.AddUint64(&metrics.retrySuccess, 1)

				} else {
					atomic.AddUint64(&metrics.retryFail, 1)
				}
			}
		}

		return rsp, err
	}

	return rsp, nil
}

// func (c *HTTPClient) Send(method, path string, hdrs map[string]string, body []byte) (*http.Response, []byte, error) {
// 	return c.SendVerbose(false, method, path, hdrs, body)
// }

// SendVerbose 전달된 정보를 바탕으로 요청을 만들어 설정된 서버와 통신하고 결과를 받는다.
func (c *HTTPClient) SendVerbose(metrics *HTTPMetrics, verbose bool, method, path string, hdrs map[string]string, body []byte) (*http.Response, []byte, error) {
	return c.sendVerbose(metrics, verbose, method, path, hdrs, body, false)
}

func (c *HTTPClient) sendVerbose(metrics *HTTPMetrics, verbose bool, method, path string, hdrs map[string]string, body []byte, retry bool) (*http.Response, []byte, error) {
	var rbuf io.Reader
	if body != nil {
		rbuf = bytes.NewBuffer(body)
	} else {
		rbuf = nil
	}
	req, err := http.NewRequest(method, c.RootPath+path, rbuf)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to create request. (req=%v %v, err=%v)\n", method, c.RootPath+path, err)
		}
		return nil, nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Host", c.TargetHost)
	if hdrs != nil {
		for k, v := range hdrs {
			req.Header.Set(k, v)
		}
	}

	if verbose {
		fmt.Println("====================================>>>>>>")
		b, _ := httputil.DumpRequest(req, true)
		fmt.Println(string(b))
		fmt.Println("")
	}

	begin := time.Now()
	rsp, err := c.Client.Do(req)
	if err != nil {

		c.reconnect()

		if retry == false {
			if verbose {
				fmt.Printf("Failed to send request. (req=%v %v, err=%v)\n", method, c.RootPath+path, err)
			}

			end := time.Now()

			diff := end.Sub(begin)
			if diff > 0 {

			}
			time.Sleep(timeout)

			atomic.AddUint64(&metrics.retryCount, 1)

			newcc, err := NewHTTPClient(c.TargetIP, c.TargetHost, c.LocalHost, c.CaCertsFile, c.CertFile, c.PrivateKeyFile, c.IsTLS, c.IsHTTP2)
			if err == nil {
				rsp, respbody, err := newcc.sendVerbose(metrics, verbose, method, path, hdrs, body, true)
				if err != nil {
					atomic.AddUint64(&metrics.retryFail, 1)
				} else {
					if rsp.StatusCode < 300 {
						atomic.AddUint64(&metrics.retrySuccess, 1)

					} else {
						atomic.AddUint64(&metrics.retryFail, 1)
					}
				}
				return rsp, respbody, err

			}
			return nil, nil, err
		}
		return nil, nil, err

	}
	defer rsp.Body.Close()

	if verbose {
		fmt.Println("<<<<<<====================================")
		b, _ := httputil.DumpResponse(rsp, true)
		fmt.Println(string(b))
		fmt.Println("")

	}

	rspbody, err := ioutil.ReadAll(rsp.Body)

	if err != nil {
		if verbose {
			fmt.Printf("Failed to read response body. (req=%v %v, err=%v)\n", method, c.RootPath+path, err)
		}
	}

	if retry == false && (err != nil || rsp.StatusCode >= 500) {

		end := time.Now()

		diff := end.Sub(begin)
		if timeout-diff > 0 {
			//time.Sleep(timeout - diff)
		}
		time.Sleep(timeout)

		atomic.AddUint64(&metrics.retryCount, 1)

		newcc, err := NewHTTPClient(c.TargetIP, c.TargetHost, c.LocalHost, c.CaCertsFile, c.CertFile, c.PrivateKeyFile, c.IsTLS, c.IsHTTP2)
		if err == nil {
			rsp, respbody, err := newcc.sendVerbose(metrics, verbose, method, path, hdrs, body, true)
			if err != nil {
				atomic.AddUint64(&metrics.retryFail, 1)
			} else {
				if rsp.StatusCode < 300 {
					atomic.AddUint64(&metrics.retrySuccess, 1)

				} else {
					atomic.AddUint64(&metrics.retryFail, 1)
				}
			}
			return rsp, respbody, err
		}
	}

	return rsp, rspbody, err
}
