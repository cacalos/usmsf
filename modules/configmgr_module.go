package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/configmgr"
)

type ConfigMgrModule struct {
}

func (r *ConfigMgrModule) Configure(binder *di.Binder) {
	binder.BindProvider((*configmgr.ConfigServer)(nil), func(injector di.Injector) interface{} {
		//binder.BindProvider((*configmgr.GetData)(nil), func(injector di.Injector) interface{} {
		configserver := injector.InjectAndCall(configmgr.NewConfigServer)
		if configserver == nil {
			return nil
		}
		configserver.(*configmgr.ConfigServer).Start()
		return configserver

	}).AsEagerSingleton()
}
