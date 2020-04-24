package modules

import (
	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/usmsf.git/implements/disabletoken"
	"github.com/csgura/di"
)

/* This Modules is NRF AccessToken Request Disabled */
type NRFAccessTokenDisableModule struct {
}

/* 음.. 해당 모듈은 Access Token 관련 사항만 Bind 해야 하는데...
   일단은 Module 자체를 별도로 분리해서 custom-cli 쪽에 밀어 넣어 보자... */

//Configure => Bind Access Token Disable Module.
func (r *NRFAccessTokenDisableModule) Configure(binder *di.Binder) {
	binder.BindConstructor((*scpcli.NrfAccessTokenService)(nil), disabletoken.NewCustomTokenDisableMode)
}
