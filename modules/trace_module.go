package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/tcptracemgr"
	"github.com/csgura/di"
)

type TraceModule struct {
}

func (r *TraceModule) Configure(binder *di.Binder) {
	binder.BindProvider((*tcptracemgr.TraceServer)(nil), func(injector di.Injector) interface{} {

		traceserver := injector.InjectAndCall(tcptracemgr.NewTraceServer)
		if traceserver == nil {
			return nil
		}
		traceserver.(*tcptracemgr.TraceServer).Start()
		return traceserver

	}).AsEagerSingleton()
}
