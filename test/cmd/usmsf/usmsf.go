package main

import (
	"os"
	"os/signal"
	"syscall"

	"runtime"

	"camel.uangel.com/ua5g/ulib.git/hocon"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/usmsf.git/modules"
)

/*
func init() {
	uredis.Setup()
}
*/

func main() {

	env := os.Getenv("USMSF_CONFIG_FILE")

	cfg := hocon.New(env)
	maxConnCnt := cfg.GetInt("mariaDB.SetMaxOpenConns", 100)
	if maxConnCnt > runtime.GOMAXPROCS(0) {
		runtime.GOMAXPROCS(0)
		ulog.Info("Start USMSF Service(%d)", runtime.GOMAXPROCS(0))
	} else {
		ulog.Info("Start USMSF Service(%d)", runtime.GOMAXPROCS(maxConnCnt))
	}

	injector := modules.Main()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	//ulog.Info("SystemSignal : %s", <-quit)
	modules.Close(injector)
}
