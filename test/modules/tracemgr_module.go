package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/tracemgr"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

// TraceMgrModule implments MetricFactory
type TraceMgrModule struct {
}

// Configure implments AbstractModule.Configure
func (r *TraceMgrModule) Configure(binder *di.Binder) {

	binder.BindConstructor((*interfaces.TraceMgr)(nil), tracemgr.NewJaegerTraceMgr)

}
