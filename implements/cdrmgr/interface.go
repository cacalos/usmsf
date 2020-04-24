package cdr

type CdrWrite interface {
	WriteCdr(Key string, MsgType string, OrigSupi string, DestSupi string, TimeStamp string,
		DoneTime string, Result string)
	//WriteCdr()
}
