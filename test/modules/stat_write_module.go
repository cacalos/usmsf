package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/implements/controller"
)

// StatModule implments MetricFactory
type StatWriteModule struct {
}

// Configure implments AbstractModule.Configure
func (r *StatWriteModule) Configure(binder *di.Binder) {

	binder.BindProvider((*controller.NFStat)(nil), func(injector di.Injector) interface{} {
		cfg := injector.GetInstance((*uconf.Config)(nil)).(uconf.Config)
		stats := injector.GetInstance((*controller.Stats)(nil)).(*controller.Stats)

		nfserverstat := controller.NewStatWrite(cfg, stats)
		if nfserverstat == nil {
			return nil
		}

		return nfserverstat

	}).AsEagerSingleton()

}
