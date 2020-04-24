package modules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/usmsf.git/implements/healthchecker"
)

// HealthCheckerModule : Health Checker용 모듈
type HealthCheckerModule struct {
}

// Configure :
// TODO
func (r *HealthCheckerModule) Configure(binder *di.Binder) {

	binder.BindProvider((*healthchecker.HealthChecker)(nil), func(injector di.Injector) interface{} {

		healthChecker := injector.InjectAndCall(healthchecker.NewHealthChecker)
		if healthChecker == nil {
			return nil
		}

		healthChecker.(*healthchecker.HealthChecker).Start()
		return healthChecker

	}).AsEagerSingleton()
}
