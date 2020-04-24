package mockupmodules

import (
	"github.com/csgura/di"

	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/mockups"
)

// NRFSimulatorModule NRF 시뮬레이터 모듈
type NRFSimulatorModule struct {
}

// Configure AbstaractModule 인터페이스 구현
func (m *NRFSimulatorModule) Configure(binder *di.Binder) {
	binder.BindProvider((*mockups.NRFSimulator)(nil), func(injector di.Injector) interface{} {
		cfg := injector.GetInstance((*uconf.Config)(nil)).(uconf.Config)
		nrf := mockups.NewNRFSimulator(cfg)
		nrf.Start()

		return nrf
	}).AsEagerSingleton()
}
