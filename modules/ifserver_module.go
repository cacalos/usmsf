package modules

import (
	"github.com/csgura/di"

	"os"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/httpif"
)

var loggers = common.SamsungLoggers()

type HttpifModule struct {
}

func (r *HttpifModule) Configure(binder *di.Binder) {
	binder.BindProvider((*httpif.IfServer)(nil), func(injector di.Injector) interface{} {
		svctype := os.Getenv("MY_SERVICE_TYPE")

		if svctype != "" {
			if svctype == "SIGTRAN" {
				ifserver := injector.InjectAndCall(httpif.NewIfServer)
				if ifserver == nil {
					return nil
				}
				ifserver.(*httpif.IfServer).Start()
				return ifserver

			} else if svctype == "DIAMETER" {
				loggers.InfoLogger().Comment("No Redis POD Mode")
				ifserver := injector.InjectAndCall(httpif.NewIfServerDia)
				if ifserver == nil {
					return nil
				}
				ifserver.(*httpif.IfServer).Start()
				return ifserver

			} else {

			}
		} else {
			loggers.ErrorLogger().Major("Fail -> Get env Service_type")
		}

		return nil

	}).AsEagerSingleton()
}
