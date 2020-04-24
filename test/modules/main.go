package modules

import (
	"math/rand"
	"os"
	"time"

	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/ulog"

	"camel.uangel.com/ua5g/usmsf.git/implements/utils"
)

func Main() testhelper.Application {
	rand.Seed(time.Now().UnixNano())

	if os.Getenv("USMSF_HOME") == "" {
		os.Setenv("USMF_HOME", utils.GetModuleRootPath("/modules"))
	}

	cfg := testhelper.LoadConfigFromEnv("USMSF_CONFIG_FILE")

	if cfg != nil {
		ulog.Info("cfg = %s", cfg.String())
	} else {
		ulog.Error("cfg is nil")
	}
	return testhelper.MainFromConfig(cfg, GetImplements())
}

func Close(app testhelper.Application) {
	app.CloseGracefully(1 * time.Second)
}
