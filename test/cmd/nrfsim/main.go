package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"camel.uangel.com/ua5g/ulib.git/testhelper"

	"camel.uangel.com/ua5g/usmsf.git/implements/utils"
	"camel.uangel.com/ua5g/usmsf.git/modules"
)

func main() {
	injector := NRFSimMain()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	modules.Close(injector)
}

// NRFSimMain NRF 시뮬레이터 메인
func NRFSimMain() testhelper.Application {
	rand.Seed(time.Now().UnixNano())

	if os.Getenv("NRFSIM_HOME") == "" {
		os.Setenv("NRFSIM_HOME", utils.GetModuleRootPath("/cmd/nrfsim"))
	}

	cfg := testhelper.LoadConfigFromEnv("NRFSIM_CONFIG_FILE")

	return testhelper.MainFromConfig(cfg, modules.GetImplements())
}
