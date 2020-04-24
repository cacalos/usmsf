package main

import (
	"os"
	"os/signal"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/usmsf.git/modules"

	"runtime"

	"camel.uangel.com/ua5g/ulib.git/hocon"
)

func main() {

	ulog.Info("Start HTTPIF Service")

	env := os.Getenv("USMSF_CONFIG_FILE")
	cfg := hocon.New(env)
	maxprocs := cfg.GetInt("httpif.maxprocs", 100)

	if maxprocs != runtime.GOMAXPROCS(0) {
		runtime.GOMAXPROCS(maxprocs)
		ulog.Info("Start USMSF Service(%d)", runtime.GOMAXPROCS(0))
	}

	injector := modules.Main()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	modules.Close(injector)
}
