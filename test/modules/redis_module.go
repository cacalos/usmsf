package modules

import (
	"camel.uangel.com/ua5g/usmsf.git/dao"
	"camel.uangel.com/ua5g/usmsf.git/implements/db"
	"github.com/csgura/di"
)

// RedisDBModule implments AbstractModule
type RedisDBModule struct {
}

// Configure implments AbstractModule.Configure
func (r *RedisDBModule) Configure(binder *di.Binder) {

	binder.BindConstructor((*db.DBManager)(nil), db.RedisNew)

	binder.BindProvider((*dao.RedisMgr)(nil), func(injector di.Injector) interface{} {
		dbmgr := injector.GetInstance((*db.DBManager)(nil)).(*db.DBManager)
		return &dbmgr
	})

	binder.BindProvider((*dao.RedisSubDao)(nil), func(injector di.Injector) interface{} {
		dbmgr := injector.GetInstance((*db.DBManager)(nil)).(*db.DBManager)
		return db.NewRedisDbSubInfoDAO(dbmgr)
	})

	binder.BindProvider((*dao.RedisDaoSet)(nil), func(injector di.Injector) interface{} {
		daoset := dao.RedisDaoSet{}
		injector.InjectMembers(&daoset)
		return &daoset
	})
}
