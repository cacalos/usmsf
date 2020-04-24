package main

import (
	"camel.uangel.com/ua5g/ulib.git/testhelper"

	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"uangel.com/usmsf/implements/healthchecker"
	"uangel.com/usmsf/modules"
)

func TestInit(t *testing.T) {
	Convey("TestInit", t, func() {

		injector := testhelper.NewInjector("reference.conf", modules.GetImplements())
		defer injector.Close()

		/*
			var cs configmgr.ConfigServer
			injector.InjectValue(&cs)
		*/

		var hc healthchecker.HealthChecker
		injector.InjectValue(&hc)

		err := hc.Init()

		So(err, ShouldBeNil)
	})

}
