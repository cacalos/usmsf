package dao

// DaoSet
type RedisDaoSet struct {
	RedisSubDao RedisSubDao `di:"inject"`
}

type MysqlDaoSet struct {
	MySqlSubDao MySqlSubDao `di:"inject"`
}
