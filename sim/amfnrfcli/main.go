package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/usmsf.git/implements/utils"
	"camel.uangel.com/ua5g/usmsf.git/modules"
)

func main() {
	ulog.Info("Start NRFClient")

	injector := Main()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	modules.Close(injector)
}

func Main() testhelper.Application {
	rand.Seed(time.Now().UnixNano())

	if os.Getenv("AMF_NRF_HOME") == "" {
		os.Setenv("AMF_NRF_HOME", utils.GetModuleRootPath("/modules"))
	}

	cfg := testhelper.LoadConfigFromEnv("AMF_NRF_CONFIG_FILE")

	if cfg != nil {
		ulog.Info("cfg = %s", cfg.String())
	} else {
		ulog.Error("cfg is nil")
	}
	return testhelper.MainFromConfig(cfg, modules.GetImplements())
}

func Close(app testhelper.Application) {
	app.CloseGracefully(1 * time.Second)
}
