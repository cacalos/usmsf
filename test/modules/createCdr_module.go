package modules

import (
	cdr "camel.uangel.com/ua5g/usmsf.git/implements/cdrmgr"
	"github.com/csgura/di"
)

// CreateCDR Module implemnts AbstractModule
type CreateCDRModule struct {
}

func (r *CreateCDRModule) Configure(binder *di.Binder) {
	binder.BindConstructor((*cdr.CdrWrite)(nil), cdr.CreateCdrMain)
}
