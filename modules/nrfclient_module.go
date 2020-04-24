package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/implements/nrfclient"
	"github.com/csgura/di"
)

// NRFClientModule AbstractModule
type NRFClientModule struct {
}

// Configure AbstractModule Configure
func (r *NRFClientModule) Configure(binder *di.Binder) {
	binder.BindProvider((*nrfclient.NRFClient)(nil), func(injector di.Injector) interface{} {

		nrfClient := injector.InjectAndCall(nrfclient.NewNRFClient)
		if nrfClient == nil {
			return nil
		}
		/*
			cfg := injector.GetInstance((*uconf.Config)(nil)).(uconf.Config)
			traceMgr := injector.GetInstance((*interfaces.TraceMgr)(nil)).(interfaces.TraceMgr)
			nrfClient := nrfclient.NewNRFClient(cfg, traceMgr)
			nrfClient := nrfclient.NewNRFClient(cfg)
		*/
		nrfClient.(*nrfclient.NRFClient).Start()
		return nrfClient
	}).AsEagerSingleton()
}
