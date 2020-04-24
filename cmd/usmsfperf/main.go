package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"camel.uangel.com/ua5g/ulib.git/testhelper"
)

func main() {
	var params Params

	err := params.Init()
	if err != nil {
		params.Help(err)
		return
	}

	cfg := testhelper.LoadConfigFromFile(params.Conf)
	if cfg == nil {
		params.Help(fmt.Errorf("Failed to load configuratoin (conf=%v)", params.Conf))
		return
	}
	maxprocs := cfg.GetInt("GOMAXPROCS", runtime.GOMAXPROCS(0))
	if maxprocs != runtime.GOMAXPROCS(0) {
		runtime.GOMAXPROCS(maxprocs)
	}

	tester := GetLoadTester(params.Scenario)
	if tester == nil {
		params.Help(fmt.Errorf("Unknown scenario '%v'", params.Scenario))
		return
	}
	err = tester.Initialize(&params)
	if err != nil {
		params.Help(fmt.Errorf("Failed to initialize tester (scenario=%v, err=%v)", params.Scenario, err))
		return
	}

	lg := NewLoadGenerator(&params)

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		fmt.Printf("TERMINATED by SIGNAL\n")
		lg.Stop()
		//os.Exit(0)
	}()

	lg.Execute(tester)
	tester.Finalize()
}
