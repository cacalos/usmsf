package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/svctracemgr"
	"github.com/csgura/di"
)

type UTraceModule struct {
}

func (r *UTraceModule) Configure(binder *di.Binder) {
	//binder.BindProvider((*utrace.TraceChecker)(nil), func(injector di.Injector) interface{} {
	binder.BindProvider((*svctracemgr.TraceSvcPod)(nil), func(injector di.Injector) interface{} {
		SvcPodTrace := injector.InjectAndCall(svctracemgr.NewSvcPodTrace)
		if SvcPodTrace == nil {
			return nil
		}
		SvcPodTrace.(*svctracemgr.TraceSvcPod).Start()
		return SvcPodTrace

	}).AsEagerSingleton()
}
