package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/controller"
)

type NFServerModule struct {
}

func (r *NFServerModule) Configure(binder *di.Binder) {
	binder.BindProvider((*controller.NFServer)(nil), func(injector di.Injector) interface{} {

		nfserver := injector.InjectAndCall(controller.NewNFServer)
		if nfserver == nil {
			return nil
		}
		nfserver.(*controller.NFServer).Start()
		return nfserver

	}).AsEagerSingleton()
}
