package dao

// SubDao - Subscriber information interface in redis
type RedisSubDao interface {
	GetSubBySUPI(supi string) (rval int, subInfo []byte)
	InsSub(supi string, insData []byte) (rval int)
	DelSub(supi string) (rval int)
	CheckSub(supi string) (rval int)
	GetSubInfoBySUPI(supi string, field string) (rval int, subInfo []byte)
}

type RedisMgr interface {
	Publish(supi string)
}
