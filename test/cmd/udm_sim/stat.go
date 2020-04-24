package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"camel.uangel.com/ua5g/ulib.git/ustat"
	metrics "github.com/rcrowley/go-metrics"
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

	regi   ustat.Timer
	deregi ustat.Timer
	sdmget ustat.Timer
	subs   ustat.Timer
	unsubs ustat.Timer
	n1n2   ustat.Timer

	nfCounter ustat.Counter

	discoverHisto map[string]ustat.Histogram
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		success:       ustat.GetOrRegisterTimer("success", nil),
		rsptime:       ustat.GetOrRegisterTimer("rsptime", nil),
		regi:          ustat.GetOrRegisterTimer("Regi", nil),
		deregi:        ustat.GetOrRegisterTimer("Deregi", nil),
		n1n2:          ustat.GetOrRegisterTimer("N1N2", nil),
		sdmget:        ustat.GetOrRegisterTimer("SdmGet", nil),
		subs:          ustat.GetOrRegisterTimer("Subs", nil),
		unsubs:        ustat.GetOrRegisterTimer("Unsubs", nil),
		nfCounter:     ustat.GetOrRegisterCounter("NfCounter", nil),
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

func (m *HTTPMetrics) RegistrationStop(start time.Time, code int) {
	now := time.Now()
	m.regi.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) DeRegistrationStop(start time.Time, code int) {
	now := time.Now()
	m.deregi.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) N1n2Stop(start time.Time, code int) {
	now := time.Now()
	m.n1n2.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) SdmGetStop(start time.Time, code int) {
	now := time.Now()
	m.sdmget.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) SubscriptionStop(start time.Time, code int) {
	now := time.Now()
	m.subs.Update(now.Sub(start))
	m.Stop(start, code)
}

func (m *HTTPMetrics) UnSubscriptionStop(start time.Time, code int) {
	now := time.Now()
	m.unsubs.Update(now.Sub(start))
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

	PrintServiceStat("UECM-Registration", m.regi, duration)
	PrintServiceStat("UECM-Deregitsration", m.deregi, duration)
	PrintServiceStat("SDM-Get", m.sdmget, duration)
	PrintServiceStat("SDM-Subscription", m.subs, duration)
	PrintServiceStat("SDM-Unsubscription", m.unsubs, duration)
	PrintServiceStat("AMF-N1N2", m.n1n2, duration)

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
