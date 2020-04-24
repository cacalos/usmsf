package modules

import (
	"github.com/csgura/di"
	//"uangel.com/usmsf/implements/statmgr"
	//"uangel.com/usmsf/interfaces"
)

// StatModule implments MetricFactory
type StatModule struct {
}

// Configure implments AbstractModule.Configure
func (r *StatModule) Configure(binder *di.Binder) {

	//	binder.BindConstructor((*interfaces.StatMgr)(nil), statmgr.NewGoStatMgr)

}
