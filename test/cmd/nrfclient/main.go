package main

import (
	"os"
	"os/signal"
	"syscall"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/usmsf.git/modules"
)

func main() {
	ulog.Info("Start NRFClient")

	injector := modules.Main()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	modules.Close(injector)
}
