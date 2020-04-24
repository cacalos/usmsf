package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/dao"
	"camel.uangel.com/ua5g/usmsf.git/implements/db"
	"github.com/csgura/di"
)

// DBModule implments AbstractModule
type MysqlDBModule struct {
}

// Configure implments AbstractModule.Configure
func (r *MysqlDBModule) Configure(binder *di.Binder) {

	binder.BindConstructor((*db.MariaDBMgr)(nil), db.MariaNew)

	binder.BindProvider((*dao.MySqlSubDao)(nil), func(injector di.Injector) interface{} {
		dbmgr := injector.GetInstance((*db.MariaDBMgr)(nil)).(*db.MariaDBMgr)
		return db.NewMariaDbSubInfoDAO(dbmgr)
	})

	binder.BindProvider((*dao.MysqlDaoSet)(nil), func(injector di.Injector) interface{} {
		daoset := dao.MysqlDaoSet{}
		injector.InjectMembers(&daoset)
		return &daoset
	})
}
