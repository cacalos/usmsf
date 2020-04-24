package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/configmgr"
	"github.com/csgura/di"
)

type UccmsClientModule struct {
}

func (r *UccmsClientModule) Configure(binder *di.Binder) {
	binder.BindProvider((*configmgr.GetData)(nil), func(injector di.Injector) interface{} {
		configClient := injector.InjectAndCall(configmgr.NewConfigClient)
		if configClient == nil {
			return nil
		}

		return configClient

	}).AsEagerSingleton()
}
