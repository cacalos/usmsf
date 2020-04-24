package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Params Parameter 정보
type Params struct {
	Conf         string
	Scenario     string
	Duration     int
	Rates        int
	ReportPeriod int
	ThreadCount  int
	ClientCount  int
	Priority     string
	Verbose      bool
	supi         string
}

var timeout = 3 * time.Second

// Init Parameter를 초기화한다.
func (p *Params) Init() error {
	conf := flag.String("f", "application.conf", "test configuration")
	scenario := flag.String("s", "", "test scenario")
	duration := flag.Int("d", 1, "duation - test period (seconds)")
	rates := flag.Int("n", 1, "call count per second(rates)")
	reportPeriod := flag.Int("p", 1, "report period")
	nthreads := flag.Int("t", runtime.NumCPU(), "thread count")
	nclient := flag.Int("c", 1, "client count")
	priority := flag.String("pri", "", "messge priority")
	verbose := flag.Bool("v", false, "messge priority")
	tmo := flag.Int("timeout", 3, "request timeout")
	supi := flag.String("supi", "450040000000001", "request supi")

	timeout = time.Duration(*tmo) * time.Second

	flag.Usage = p.Usage
	flag.Parse()
	if scenario == nil || *scenario == "" {
		return fmt.Errorf("No argument - scenario")
	}
	p.Conf = *conf
	p.Scenario = *scenario
	p.Duration = *duration
	p.Rates = *rates
	p.ReportPeriod = *reportPeriod
	p.ThreadCount = *nthreads
	p.ClientCount = *nclient
	p.Priority = *priority
	p.Verbose = *verbose
	p.supi = *supi

	if p.ClientCount <= 0 {
		p.ClientCount = 1
	}
	return nil
}

// Help 도움말을 출력한다.
func (p *Params) Help(err error) {
	if err != nil && err != flag.ErrHelp {
		fmt.Printf("%v\n", err)
	}
	p.Usage()
}

// Usage 도움말을 출력한다.
func (p *Params) Usage() {
	fname := os.Args[0]
	if fname == "" {
		fmt.Printf("Usage:\n")
	} else {
		fmt.Printf("Usage of %s:\n", fname)
	}
	flag.PrintDefaults()
	fmt.Printf("\n[Secenarios]\n")
	testers := GetLoadTesterNames()
	if testers != nil {
		for i, n := range testers {
			fmt.Printf("   [%2d] %s\n", i, n)
		}
	}
}
