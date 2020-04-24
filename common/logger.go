package common

import "camel.uangel.com/ua5g/ulib.git/ulog"

var samsungLoggers = ulog.SamsungLoggers{}
var isSamsungLoggersInitialized = false

// SamsungLoggers : 삼성향 logger set을 반환한다.
func SamsungLoggers() *ulog.SamsungLoggers {

	if !isSamsungLoggersInitialized {
		initSamsungLogger(&samsungLoggers)
		isSamsungLoggersInitialized = true
	}

	return &samsungLoggers
}

func initSamsungLogger(loggers *ulog.SamsungLoggers) {

	loggers.InitFaultLogger("samsung.fault")
	loggers.InitSelfDiagLogger("samsung.selfdiag")
	loggers.InitResetLogger("samsung.reset")
	loggers.InitInitLogger("samsung.init")
	loggers.InitConfigLogger("samsung.config")
	loggers.InitErrorLogger("samsung.error")
	loggers.InitEventLogger("samsung.event")
	loggers.InitInfoLogger("samsung.info")
}
