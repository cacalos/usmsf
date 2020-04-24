package modules

import (
	scpmodule "camel.uangel.com/ua5g/scpcli.git/modules"

	//subsmgrmodule "camel.uangel.com/ua5g/ubsmgr.git/modules"
	ulibmodule "camel.uangel.com/ua5g/ulib.git/modules"
	"camel.uangel.com/ua5g/usmsf.git/modules/mockupmodules"
	"github.com/csgura/di"
)

func GetImplements() *di.Implements {
	impls := di.NewImplements()

	//	impls.AddImplements(subsmgrmodule.GetImplements())

	//	impls.AddImplement("StatMgr", &StatModule{})
	impls.AddImplement("TraceMgr", &TraceMgrModule{})
	impls.AddImplement("NFServerModule", &NFServerModule{})
	impls.AddImplement("MapServerModule", &MapServerModule{})
	impls.AddImplement("Controller", &ControllerModule{})
	impls.AddImplement("RedisDBModule", &RedisDBModule{})
	impls.AddImplement("MysqlDBModule", &MysqlDBModule{})
	impls.AddImplement("StatWriteModule", &StatWriteModule{})
	impls.AddImplement("NRFClient", &NRFClientModule{})
	impls.AddImplement("UTraceModule", &UTraceModule{})
	//impls.AddImplement("TraceWriteModule", &TraceWriteModule{}) // 주석

	impls.AddImplement("HttpifModule", &HttpifModule{})
	impls.AddImplement("ConfigMgrModule", &ConfigMgrModule{})
	impls.AddImplement("UccmsClientModule", &UccmsClientModule{})
	impls.AddImplement("TraceModule", &TraceModule{})
	impls.AddImplement("TcpMgrModule", &TcpMgrModule{})

	smsfDefaults := di.CombineModule(
		//		&StatModule{},
		&TraceMgrModule{},
		&UTraceModule{},
	)

	httpifDefaults := di.CombineModule(
		&RedisDBModule{},
		&TraceMgrModule{},
		&ConfigMgrModule{},
		&TcpMgrModule{},
		&HttpifModule{},
		&TraceModule{},
	)

	impls.AddImplement("ControllerDefaults", di.OverrideModule(smsfDefaults))
	impls.AddImplement("httpifDefaults", di.OverrideModule(httpifDefaults))

	impls.AddImplement("UsmsfDefaults", di.OverrideModule(di.CombineModule(
		smsfDefaults,
		&NFServerModule{},
		&MapServerModule{},
		&ControllerModule{}, // 이름 바꿔야겠네..
		&RedisDBModule{},
		&MysqlDBModule{},
		&StatWriteModule{},
		//		&TraceWriteModule{},
		&ConfigMgrModule{},
	)))

	impls.AddImplements(ulibmodule.GetImplements())
	impls.AddImplements(scpmodule.GetImplements())

	impls.AddImplement("DisableAccessTokenRequestToNRF", &NRFAccessTokenDisableModule{})
	impls.AddImplement("NRFSimulator", &mockupmodules.NRFSimulatorModule{})
	impls.AddImplement("HealthCheckerModule", &HealthCheckerModule{})

	return impls
}
