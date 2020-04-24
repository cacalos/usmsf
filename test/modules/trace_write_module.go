package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/svctracemgr"
)

// StatModule implments MetricFactory
type TraceWriteModule struct {
}

// Configure implments AbstractModule.Configure
func (r *TraceWriteModule) Configure(binder *di.Binder) {

	binder.BindProvider((*svctracemgr.NFTrace)(nil), func(injector di.Injector) interface{} {
		trace := injector.GetInstance((*svctracemgr.TraceSvcPod)(nil)).(*svctracemgr.TraceSvcPod)

		nfservertrace := svctracemgr.NewTraceWrite(trace)
		if nfservertrace == nil {
			return nil
		}

		return nfservertrace

	}).AsEagerSingleton()

}
