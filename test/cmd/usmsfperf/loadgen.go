package main

import (
	"fmt"
	"math"
	"sync"
	"time"

	"camel.uangel.com/ua5g/ulib.git/ustat"
	"github.com/panjf2000/ants/v2"
)

// LoadTester 부하 테스터 인터페이스
type LoadTester interface {
	//Prepare 테스트를 실행하기 전에 시험을 준비해야 할 것들을 수행한다.
	Initialize(params *Params) error
	//Finalize 테스트를 완료한다.
	Finalize()
	//Execute 테스트를 실행한다.
	Execute() error
	//Report 테스터 결과를 보고한다. Final이 아닐경우 중간 보고이다.
	Report(isFinal bool)
}

// LoadGenerator 부하 발생기
type LoadGenerator struct {
	duration     int // 성능 시험 수행 시간 (초 단위)
	rates        int // 초당 부하율 - 초마다 테스터를 이 값만큰 실행한다.
	reportPeriod int // 테스터 report 호출 주기 (초단위)
	nthreads     int // thread count
	waitGroup    sync.WaitGroup
	termchnl     chan bool
	thrpool      *ants.PoolWithFunc
}

// countLoadTeser 시험용 실행 count를 계산하는 Reporter
type countLoadTester struct {
	count ustat.Meter
	start time.Time
}

////////////////////////////////////////////////////////////////////////////////
// Functions of LoadTester
////////////////////////////////////////////////////////////////////////////////

// LoadTester
var testers map[string]LoadTester

// RegisterLoadTester 전달된 이름으로 LoadTester를 등록한다.
func RegisterLoadTester(name string, tester LoadTester) {
	if testers == nil {
		testers = make(map[string]LoadTester)
	}
	testers[name] = tester
}

// GetLoadTester 전달된 이름의 LoadTester를 반환한다.
func GetLoadTester(name string) LoadTester {
	if testers == nil {
		return nil
	}
	return testers[name]
}

// GetLoadTesterNames LoadTester 목록를 반환한다.
func GetLoadTesterNames() []string {
	var rslt []string
	if testers == nil {
		return rslt
	}
	for n := range testers {
		rslt = append(rslt, n)
	}
	return rslt
}

////////////////////////////////////////////////////////////////////////////////
// Functions of LoadGenerator
////////////////////////////////////////////////////////////////////////////////

// NewLoadGenerator Load Generator를 실행한다.
func NewLoadGenerator(params *Params) *LoadGenerator {
	g := &LoadGenerator{
		duration:     params.Duration,
		rates:        params.Rates,
		reportPeriod: params.ReportPeriod,
		nthreads:     params.ThreadCount,
		waitGroup:    sync.WaitGroup{},
		termchnl:     make(chan bool),
	}
	if params.ThreadCount > 0 {
		thrpool, _ := ants.NewPoolWithFunc(params.ThreadCount, func(i interface{}) {
			defer g.waitGroup.Done()
			tester := i.(LoadTester)
			err := tester.Execute()
			if err != nil {
				go g.Stop()
			}
		})
		g.thrpool = thrpool
	}
	return g
}

// Release Load Generator를 종료한다.
func (g *LoadGenerator) Release() {
	if g.thrpool != nil {
		g.thrpool.Release()
	}
}

// Stop Load Generator 구동을 멈춘다.
func (g *LoadGenerator) Stop() {
	g.termchnl <- true
}

func (g *LoadGenerator) executeTester(tester LoadTester) {
	g.waitGroup.Add(1)
	go func() {
		defer g.waitGroup.Done()
		err := tester.Execute()
		if err != nil {
			go g.Stop()
		}
	}()
}

// Execute 전달된 Test를 설정된 주기와 부하로 Test를 실행한다.
func (g *LoadGenerator) Execute(tester LoadTester) {
	if g.thrpool != nil {
		g.executeTestByPool(tester)
	} else {
		g.executeEachTestByGoroutine(tester)
	}
}

// executeEachTestByGoroutine 전달된 Test를 Gourtine을 사용해 부하 발생 시킴
func (g *LoadGenerator) executeEachTestByGoroutine(tester LoadTester) {
	reportTick := 0
	nticks := 0
	if nticks < g.duration {
		for i := 0; i < g.rates; i++ {
			g.executeTester(tester)
		}
	}
	nticks++
	reportTick++
	if nticks >= g.duration {
		go g.Stop()
	}
	for {
		select {
		case <-time.After(1 * time.Second):
			if nticks < g.duration {
				for i := 0; i < g.rates; i++ {
					g.executeTester(tester)
				}
				nticks++
				reportTick++
			}
			if nticks >= g.duration {
				go g.Stop()
				break
			}
			if reportTick >= g.reportPeriod {
				reportTick = 0
				go tester.Report(false)
			}
		case <-g.termchnl:
			g.waitGroup.Wait()
			tester.Report(true)
			return
		}
	}
}

// executeTestByPool 전달된 Test를 Gourtine pool을 사용해 부하 발생 시킴
func (g *LoadGenerator) executeTestByPool(tester LoadTester) {
	start := time.Now()
	reportSec := start.Unix()

	divider := 100
	sleepTime := 10

	ratesPer := int(math.Ceil(float64(g.rates) / float64(divider)))

	for ratesPer < 10 {
		if ratesPer >= g.rates {
			break
		}
		divider = divider / 2
		sleepTime = sleepTime * 2
		ratesPer = int(math.Ceil(float64(g.rates) / float64(divider)))
	}

	if ratesPer >= g.rates {
		sleepTime = 1000
	}
	fmt.Printf("Start Load Generator. Rates %d per %d ms\n\n", ratesPer, sleepTime)

	var expected time.Time
	var delayed time.Duration
	for {
		now := time.Now()

		if !expected.IsZero() {
			delayed = now.Sub(expected)
			//fmt.Printf("delayed = %s\n", delayed)
		}
		nowSecond := now.Unix()
		if int(now.Sub(start)/time.Second) < g.duration {
			//fmt.Printf("expected =%d, now =%d , delayed = %s\n", expected.Nanosecond()/1000000, now.Nanosecond()/1000000, delayed)
			for i := 0; i < ratesPer; i++ {
				g.waitGroup.Add(1)
				g.thrpool.Invoke(tester)
			}
		} else {
			go g.Stop()
			break
		}

		if int(nowSecond-reportSec) >= g.reportPeriod {
			reportSec = nowSecond
			go tester.Report(false)
		}

		afterSubmit := time.Now()
		st := time.Duration(sleepTime)*time.Millisecond - afterSubmit.Sub(now) - delayed
		if st > 0 {
			expected = afterSubmit.Add(st)

			//fmt.Printf("st = %s\n", st)
			select {
			case <-time.After(st):
				break
			case <-g.termchnl:
				g.waitGroup.Wait()
				tester.Report(true)
				return
			}
		} else {
			//fmt.Printf("don't sleep\n")
			expected = now
			select {
			case <-g.termchnl:
				g.waitGroup.Wait()
				tester.Report(true)
				return
			default:
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Functions of countLoadTeser
////////////////////////////////////////////////////////////////////////////////

//Initialize 테스트를 실행하기 전에 시험을 준비해야 할 것들을 수행한다.
func (t *countLoadTester) Initialize(params *Params) error {
	t.count = ustat.GetOrRegisterMeter("count", nil)
	t.start = time.Now()
	return nil
}

//Finalize 테스트를 완료한다.
func (t *countLoadTester) Finalize() {
	//Do Nothing
}

//Execute 테스트를 실행한다.
func (t *countLoadTester) Execute() error {
	//atomic.AddUint64(&t.count, 1)
	t.count.Mark(1)
	time.Sleep(3 * time.Millisecond)
	return nil
}

//Report 테스터 결과를 보고한다. Final이 아닐경우 중간 보고이다.
func (t *countLoadTester) Report(isFinal bool) {
	duration := float64(time.Now().Sub(t.start)) / float64(time.Second)
	fmt.Printf("[%s] count = %.2f , 1m tps = %.2f\n", time.Now().Format(time.RFC3339), float64(t.count.Count())/duration, t.count.Rate1())
}

func init() {
	RegisterLoadTester("count", &countLoadTester{})
}
