package db

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	rejson "github.com/nitishm/go-rejson"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/dao"
)

type SubInfoDAO struct {
	Client         *redis.Client
	ExpireDuration time.Duration
	DB             *gorm.DB
}

type DBManager struct {
	PubSubClient   *redis.Client
	Client         *redis.Client
	PubSub         *redis.PubSub
	SubInfoDao     *SubInfoDAO
	redisHost      string
	pubsubHost     string
	channel        string
	ExpireDuration time.Duration
}

type MariaDBMgr struct {
	ManageDB  *gorm.DB
	ContextDB *gorm.DB
}

const (
	DBFail    = -1
	DBSuccess = 1
)

var loggers = common.SamsungLoggers()

const (
	// StatusDBNotFound - DB Record Not Found.
	StatusDBNotFound = 404
	// StatusDBFail - Query
	StatusDBFail = 598
	// StatusDBConnError - DB Connection Pool
	StatusDBConnError = 599
)

// DBErrorCode error ErrorWithCode Use
func DBErrorCode(err error) error {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case (*errcode.ErrorWithCode):
		return err
	default:
		if gorm.IsRecordNotFoundError(err) {
			return errcode.WithCode(err, StatusDBNotFound, "DB Record Not Found")
		} else if strings.Contains(err.Error(), "commands out of sync") {
			return errcode.WithCode(err, StatusDBConnError, "DB Connection Error")
		}
		return errcode.WithCode(err, StatusDBFail, "DB Error")
	}
}

// Redis DB Manager New
func RedisNew(cfg uconf.Config) *DBManager {
	dbmgr, err := DBManager{}.Init(cfg)
	if err != nil {
		dbmgr = &DBManager{}
		loggers.ErrorLogger().Major("%v", err)
		return dbmgr
	}
	return dbmgr
}

// Maria DB Manager New
func MariaNew(cfg uconf.Config) *MariaDBMgr {
	dbmgr, err := MariaDBMgr{}.Connect(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}
	return dbmgr
}

// Connect MariaDB
func (r MariaDBMgr) Connect(cfg uconf.Config) (*MariaDBMgr, error) {
	var err error

	// To do ConfigFile
	driver := cfg.GetString("mariaDB.driver")
	user := cfg.GetConfig("mariaDB.user")
	pwd := cfg.GetConfig("mariaDB.pwd")
	host := cfg.GetConfig("mariaDB.server.uri")
	dbname := cfg.GetConfig("mariaDB.dbname")

	SetMaxIdleConns := cfg.GetInt("mariaDB.SetMaxIdleConns", 100)
	SetMaxOpenConns := cfg.GetInt("mariaDB.SetMaxOpenConns", 100)
	SetConnMaxLifetime := cfg.GetDuration("mariaDB.SetConnMaxLifetime", 5)

	spec := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, pwd, host, dbname)

	loggers.InfoLogger().Comment("driver : %s, user : %s, pwd : %s, host : %s, dbname : %s, spec : %s", driver, user, pwd, host, dbname, spec)
	r.ManageDB, err = gorm.Open(driver, spec)
	if err != nil {
		loggers.ErrorLogger().Major("Mysql Connection Error")
		return nil, err
	}

	r.ManageDB.SetLogger(loggers.ErrorLogger().ULogger())

	r.ManageDB.DB().SetConnMaxLifetime(time.Minute * SetConnMaxLifetime)
	r.ManageDB.DB().SetMaxIdleConns(SetMaxIdleConns)
	r.ManageDB.DB().SetMaxOpenConns(SetMaxOpenConns)

	loggers.InfoLogger().Comment("MariaDBMgr Connect Success !! ")

	return &r, nil
}

// Maria DB Manager New
func UCCMSMariaNew(cfg uconf.Config) *MariaDBMgr {
	dbmgr, err := MariaDBMgr{}.UccmsConnect(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}
	return dbmgr
}

// Connect MariaDB for UCCMS(instance)
func (r MariaDBMgr) UccmsConnect(cfg uconf.Config) (*MariaDBMgr, error) {
	var err error

	// To do ConfigFile
	driver := cfg.GetString("InstanceUCCMSDB.driver")
	user := cfg.GetConfig("InstanceUCCMSDB.user")
	pwd := cfg.GetConfig("InstanceUCCMSDB.pwd")
	host := cfg.GetConfig("InstanceUCCMSDB.server.uri")
	dbname := cfg.GetConfig("InstanceUCCMSDB.dbname")

	SetMaxIdleConns := cfg.GetInt("mariaDB.SetMaxIdleConns", 100)
	SetMaxOpenConns := cfg.GetInt("mariaDB.SetMaxOpenConns", 100)
	SetConnMaxLifetime := cfg.GetDuration("mariaDB.SetConnMaxLifetime", 5)

	spec := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, pwd, host, dbname)

	loggers.InfoLogger().Comment("driver : %s, user : %s, pwd : %s, host : %s, dbname : %s, spec : %s", driver, user, pwd, host, dbname, spec)
	r.ManageDB, err = gorm.Open(driver, spec)
	if err != nil {
		loggers.ErrorLogger().Major("Mysql Connection Error")
		return nil, err
	}

	r.ManageDB.SetLogger(loggers.ErrorLogger().ULogger())

	r.ManageDB.DB().SetConnMaxLifetime(time.Minute * SetConnMaxLifetime)
	r.ManageDB.DB().SetMaxIdleConns(SetMaxIdleConns)
	r.ManageDB.DB().SetMaxOpenConns(SetMaxOpenConns)

	loggers.InfoLogger().Comment("MariaDBMgr Connect Success !! ")

	return &r, nil
}

func NewMariaDbSubInfoDAO(mgr *MariaDBMgr) *SubInfoDAO {
	return &SubInfoDAO{DB: mgr.ManageDB}
}

// Close MariaDB Manager
func (r *MariaDBMgr) Final() {
	if r.ContextDB != r.ManageDB {
		r.ContextDB.Close()
	}
	r.ManageDB.Close()
}

func (r DBManager) Listen() {

	var channel, payload string

	for {
		msg, err := r.PubSub.ReceiveTimeout(time.Second)
		if err != nil {
			// Timeout, ignore
			if reflect.TypeOf(err) == reflect.TypeOf(&net.OpError{}) && reflect.TypeOf(err.(*net.OpError).Err).String() == "*net.timeoutError" {
				continue
			}
			time.Sleep(1 * time.Second)
			continue
		}

		switch m := msg.(type) {
		case *redis.Subscription:
			continue
		case *redis.Message:
			channel = m.Channel
			payload = m.Payload
		default:
			continue
		}

		loggers.InfoLogger().Comment("Recevie Channel:%s, Msg:%s", channel, payload)

		amfsupi := fmt.Sprintf("amf-%s", payload)
		sdmsupi := fmt.Sprintf("sdm-%s", payload)

		r.SubInfoDao.DelSub(amfsupi)
		r.SubInfoDao.DelSub(sdmsupi)
	}

}

func (r DBManager) RedisSessionCheck() {
	var ticker *time.Ticker

	for {

		if ticker == nil {
			ticker = time.NewTicker(time.Second * 60)
		}

		select {
		case <-ticker.C:
			if err := r.Client.Ping().Err(); err != nil {
				loggers.ErrorLogger().Major("Redis Ping Error " + err.Error())
				r.Client = redis.NewClient(&redis.Options{Addr: r.redisHost})
				if err := r.Client.Ping().Err(); err != nil {
					loggers.ErrorLogger().Major("Unable to connect to redis " + err.Error())
				}
				ticker.Stop()
				ticker = nil
			}
		default:
			time.Sleep(time.Second * 10) // Idle Timer
			continue
		}
	}
}

func (r DBManager) RedisSubSessionCheck() {
	var ticker *time.Ticker

	for {

		if ticker == nil {
			ticker = time.NewTicker(time.Second * 60)
		}

		select {
		case <-ticker.C:
			if err := r.PubSubClient.Ping().Err(); err != nil {
				loggers.ErrorLogger().Major("Redis Ping Error " + err.Error())
				r.PubSubClient = redis.NewClient(&redis.Options{Addr: r.pubsubHost})
				if err := r.PubSubClient.Ping().Err(); err != nil {
					loggers.ErrorLogger().Major("Unable to connect to redis " + err.Error())
				} else {
					r.PubSub = r.PubSubClient.Subscribe(r.channel)
				}
				ticker.Stop()
				ticker = nil
			}
		default:
			time.Sleep(time.Second * 10) // Idle Timer
			continue
		}
	}
}

func (r DBManager) Publish(supi string) {

	err := r.PubSubClient.Publish(r.channel, supi)

	if err != nil {
		loggers.ErrorLogger().Major("Redis Pulish Error : %s", err)
	} else {
		loggers.InfoLogger().Comment("Redis PUBLISH Channel : %s, MSG : %s", r.channel, supi)
	}
}

// Connect RedisDB
func (r DBManager) Init(cfg uconf.Config) (*DBManager, error) {

	var redisHost, pubsubHost, mychannel string

	redisCfg := cfg.GetConfig("redis")
	if redisCfg != nil {
		redisHost = redisCfg.GetString("server.uri", "localhost:6379")
		pubsubHost = redisCfg.GetString("pubsub.server.uri", "localhost:6379")
		mychannel = redisCfg.GetString("channel", "smsf")
	}

	r.redisHost = redisHost

	r.Client = redis.NewClient(&redis.Options{Addr: redisHost})
	if err := r.Client.Ping().Err(); err != nil {
		loggers.ErrorLogger().Major("Unable to connect to redis " + err.Error())
		return nil, err
	}

	r.pubsubHost = pubsubHost
	r.channel = mychannel

	r.PubSubClient = redis.NewClient(&redis.Options{Addr: pubsubHost})
	if err := r.PubSubClient.Ping().Err(); err != nil {
		loggers.ErrorLogger().Major("Unable to connect to redis " + err.Error())
		return nil, err
	} else {
		r.PubSub = r.PubSubClient.Subscribe(mychannel)
	}

	subInfoDao := &SubInfoDAO{
		Client:         r.Client,
		ExpireDuration: r.ExpireDuration,
	}

	r.SubInfoDao = subInfoDao

	timeVal := redisCfg.GetDuration("subs-expireation", time.Second*3600)
	r.ExpireDuration = timeVal / 1000 / 1000 / 1000

	loggers.InfoLogger().Comment("Redis Connection Info, HOST[%s], subs expiration[%d]", redisHost, r.ExpireDuration)

	exec.SafeGo(r.RedisSessionCheck)

	exec.SafeGo(r.RedisSubSessionCheck)

	exec.SafeGo(r.Listen)

	return &r, nil
}

func NewRedisDbSubInfoDAO(mgr *DBManager) *SubInfoDAO {
	return &SubInfoDAO{
		Client:         mgr.Client,
		ExpireDuration: mgr.ExpireDuration,
	}
}

func (o *SubInfoDAO) GetSubBySUPI(key string) (int, []byte) {

	result := make(chan int, 1)
	subChan := make(chan []byte, 1)
	exec.SafeGo(func() {
		rh := rejson.NewReJSONHandler()
		rh.SetGoRedisClient(o.Client)

		loggers.InfoLogger().Comment("Redis API Get Service, USER : %s", key)

		info, err := rh.JSONGet(key, ".")
		if err != nil {
			loggers.InfoLogger().Comment("JSON.Get Fail, USER : %s", key)
			result <- DBFail
		} else {
			result <- DBSuccess
			subChan <- info.([]byte)
		}

		close(result)
		close(subChan)
	})

	info := <-subChan
	rval1 := <-result

	return rval1, info

}

func (o *SubInfoDAO) InsSub(key string, insData []byte) int {

	var rval int

	rh := rejson.NewReJSONHandler()
	if rh == nil {
		loggers.ErrorLogger().Critical("Failed to Connect Redis Server")
		return DBFail
	}

	rh.SetGoRedisClient(o.Client)

	loggers.InfoLogger().Comment("Redis API Insert, USER : %s", key)
	loggers.InfoLogger().Comment("Redis API Insert, Data : %s", insData)

	res, err := rh.JSONSet(key, ".", insData)
	if err != nil {
		loggers.ErrorLogger().Major("Failed to Insert Data Redis Server, key:%s", key)
		rval = DBFail
	} else {
		if res.(string) == "OK" {
			loggers.InfoLogger().Comment("Insert Success: %s", res)
			o.Client.Expire(key, o.ExpireDuration*time.Second)
			rval = DBSuccess
		} else {
			loggers.ErrorLogger().Major("Insert Failed : %s", res)
			rval = DBFail
		}
	}

	return rval
}

func (o *SubInfoDAO) DelSub(key string) int {

	var rval int

	rh := rejson.NewReJSONHandler()
	rh.SetGoRedisClient(o.Client)

	loggers.InfoLogger().Comment("Redis API Delete, USER : %s", key)

	res, err := rh.JSONDel(key, ".")
	if err != nil {
		loggers.ErrorLogger().Major("Fail to Delete Data Redis Server, USER:%s", key)
		rval = DBFail
	}

	if res.(int64) == 0 {
		loggers.ErrorLogger().Major("Redis Delete fail, USER : %s", key)
		rval = DBFail
	} else {
		loggers.InfoLogger().Comment("Redis Delete success, USER : %s", key)
		rval = DBSuccess
	}

	return rval
}

func (o *SubInfoDAO) CheckSub(key string) int {

	result := make(chan int, 1)

	rh := rejson.NewReJSONHandler()
	rh.SetGoRedisClient(o.Client)

	loggers.InfoLogger().Comment("Redis API Check, USER : %s", key)

	_, err := rh.JSONGet(key, ".")
	if err != nil {
		loggers.InfoLogger().Comment("Doesn't exist in Redis DB, USER : %s", key)
		result <- DBFail
	} else {
		result <- DBSuccess
	}
	close(result)

	rval := <-result

	return rval
}

func (o *SubInfoDAO) GetSubInfoBySUPI(key string, field string) (int, []byte) {

	result := make(chan int, 1)
	subChan := make(chan []byte, 1)
	exec.SafeGo(func() {
		rh := rejson.NewReJSONHandler()
		rh.SetGoRedisClient(o.Client)

		loggers.InfoLogger().Comment("Redis API Get Service, USER : %s, Field : %s", key, field)

		info, err := rh.JSONGet(key, field)
		if err != nil {
			loggers.InfoLogger().Comment("JSON.Get Fail, USER : %s", key)
			result <- DBFail
		} else {
			result <- DBSuccess
			subChan <- info.([]byte)
		}

		close(result)
		close(subChan)
	})

	info := <-subChan
	rval1 := <-result

	return rval1, info
}

// Insert Data
func (o *SubInfoDAO) Create(row *dao.MariaInfo) {
	loggers.InfoLogger().Comment("MariaDB API Insert Service")

	o.DB.DB().Ping()
	err := o.DB.Create(row).Error
	if err != nil {
		loggers.ErrorLogger().Major("Failed to Insert Data MariaDB Server Err : %s", DBErrorCode(err))
	} else {
		loggers.InfoLogger().Comment("MariaDB API Create Success")
	}

	return
}

// Select By key for Column Existing
func (o *SubInfoDAO) ExistMariaSub(key string, row *dao.MariaInfo) int {
	loggers.InfoLogger().Comment("MariaDB API Exist Supi Service : %s", key)
	result := make(chan int, 1)

	o.DB.DB().Ping()

	err := o.DB.Where("IMSI = ?", key).Find(row).Error
	if err != nil {
		loggers.InfoLogger().Comment("Doesn't exist in MariaDB Err : %s", DBErrorCode(err))
		result <- DBFail
	} else {
		loggers.InfoLogger().Comment("MariaDB API Exist Success")
		result <- DBSuccess
	}
	close(result)

	rval := <-result

	return rval
}

// Select By key for Get Data Column
func (o *SubInfoDAO) GetSubInfoByKEY(key string) (int, []byte) {
	loggers.InfoLogger().Comment("MariaDB API Get Data Service : %s", key)
	var dbInfo dao.MariaInfo
	result := make(chan int, 1)
	subChan := make(chan []byte, 1)

	// To do Table Name ConfigFile
	o.DB.DB().Ping()
	err := o.DB.Table("maria_infos").Select("`DATA`").Where("IMSI = ?", key).Scan(&dbInfo).Error
	if err != nil {
		loggers.InfoLogger().Comment("Doesn't exist in MariaDB Err : %s", DBErrorCode(err))
		result <- DBFail
	} else {
		loggers.InfoLogger().Comment("MariaDB API GetData Success")
		result <- DBSuccess
		subChan <- dbInfo.DATA
	}

	close(result)
	close(subChan)

	dbInfo.DATA = <-subChan
	rval1 := <-result

	return rval1, dbInfo.DATA
}

// Delete By Key
func (o *SubInfoDAO) Delete(key string) {
	loggers.InfoLogger().Comment("MariaDB API Delete Supi Service : %s", key)

	o.DB.DB().Ping()
	err := o.DB.Where("IMSI = ?", key).Delete(dao.MariaInfo{}).Error
	if err != nil {
		loggers.ErrorLogger().Major("Fail to Delete Data MariaDB Server Err : :%s", DBErrorCode(err))
	} else {
		loggers.InfoLogger().Comment("MariaDB API Delete Success")
	}

	return
}

// Delete By Key for UCCMS
func (o *SubInfoDAO) UccmsDelete() error {
	loggers.InfoLogger().Comment("Delete WatchId in UCCMS database")

	o.DB.DB().Ping()
	err := o.DB.Table("watch_info").Delete("`id = ?`", dao.UccmsWatchId{}).Error
	if err != nil {
		loggers.ErrorLogger().Major("Fail to Delete Data MariaDB Server Err : :%s", DBErrorCode(err))
	} else {
		loggers.InfoLogger().Comment("MariaDB API Delete Success")
	}

	return err
}
