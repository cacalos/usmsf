package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/tcpmgr"
	"os"
)

type TcpMgrModule struct {
}

func (r *TcpMgrModule) Configure(binder *di.Binder) {
	binder.BindProvider((*tcpmgr.TcpServer)(nil), func(injector di.Injector) interface{} {

		svctype := os.Getenv("MY_SERVICE_TYPE")

		if svctype != "" {
			if svctype == "SIGTRAN" {

				tcpserver := injector.InjectAndCall(tcpmgr.NewTcpServer)
				if tcpserver == nil {
					return nil
				}
				//		tcpserver.(*tcpmgr.TcpServer).Start()
				return tcpserver

			} else if svctype == "DIAMETER" {
				loggers.InfoLogger().Comment("No Redis POD Mode")
				tcpserver := injector.InjectAndCall(tcpmgr.NewTcpServerDia)
				if tcpserver == nil {
					return nil
				}
				//		tcpserver.(*tcpmgr.TcpServer).Start()
				return tcpserver

			} else {

			}
		} else {
			loggers.ErrorLogger().Major("Fail -> Get env Service_type")
		}

		return nil

	}).AsEagerSingleton()
}
