package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/controller"
	"github.com/csgura/di"
)

type MapServerModule struct {
}

func (r *MapServerModule) Configure(binder *di.Binder) {
	binder.BindProvider((*controller.MapServer)(nil), func(injector di.Injector) interface{} {
		mapserver := injector.InjectAndCall(controller.NewMapServer)

		if mapserver == nil {
			return nil
		}
		mapserver.(*controller.MapServer).Start()
		return mapserver

	}).AsEagerSingleton()
}
