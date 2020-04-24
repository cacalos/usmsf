package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/controller"
	"github.com/csgura/di"
)

// ControllerModule implemnts AbstractModule
type ControllerModule struct {
}

// Configure implements AbstractModule.Configure
func (r *ControllerModule) Configure(binder *di.Binder) {
	binder.BindConstructor((*controller.Stats)(nil), controller.NewStats)
}
