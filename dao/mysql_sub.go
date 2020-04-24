/**
 * @brief MariaDB Column Structure & Function Interface(CRUD)
 * @author parkjh
 * @file mysql_sub.go
 * @data 2019-06-21
 * @version 0.1
 */
package dao

type MariaInfo struct {
	IMSI string `gorm:"primary_key;type:varchar(32)"`
	DATA []byte `gorm:"column:DATA;type:varchar(1024)"`
}

type UccmsWatchId struct {
	Id        int    `gorm:"primary_key;type:int(11)"`
	whtch_id  string `gorm:"unique_key;type:varchar(128)"`
	Conf_id   string `gorm:"column:conf_id;type:varchar(128)"`
	Call_back string `gorm:"column:call_back;type:varchar(1024)"`
}

// SubDao - Subscriber information interface in mysql
type MySqlSubDao interface {
	Create(row *MariaInfo)
	ExistMariaSub(key string, row *MariaInfo) int
	GetSubInfoByKEY(key string) (int, []byte)
	Delete(key string)
	UccmsDelete() error
}
