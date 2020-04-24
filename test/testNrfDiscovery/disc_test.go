package main

import (
	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/utypes"

	"uangel.com/usmsf/modules" 
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDiscovery(t *testing.T) {
	Convey("TestDiscovery", t, func() {

		injector := testhelper.NewInjector("reference.conf", modules.GetImplements())
		defer injector.Close()

		var nrfcli scpcli.NrfDiscService
		injector.InjectValue(&nrfcli)

		sr, err := nrfcli.NFDiscovery(context.Background(), "UDM", utypes.Map{})
		So(err, ShouldBeNil)
		So(sr, ShouldNotBeNil)
	})

}
