package cdr

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/usmsf.git/common"
)

// 삼성 포맷 로그들은 loggers를 사용한다.
var loggers = common.SamsungLoggers()

var SeqIndex uint32

type CdrMgr struct {
	Key       string
	MsgType   string
	OrigSupi  string
	DestSupi  string
	TimeStamp string
	DoneTime  string
	Result    string
}

func CreateCdrMain(cfg uconf.Config) *CdrMgr {

	cdr := &CdrMgr{}

	return cdr

}

func (w *CdrMgr) ExistFileName(fileName string) error {
	var path string

	check, err := get_file_size(fileName)
	if err != nil {
		path = fmt.Sprintf("%s/cdr.%s", os.Getenv("CDR_DIR"), w.TimeStamp)

		fd, err := os.OpenFile(path,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)

		if err != nil {
			loggers.ErrorLogger().Major("File Open Fail")
			return err

		}

		defer fd.Close()
		w.Record_Cdr(fd)

		return nil
	}

	if check != true {
		fd, err := os.OpenFile(fileName,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)

		if err != nil {
			loggers.ErrorLogger().Major("File Open Fail")
			return err

		}

		defer fd.Close()
		w.Record_Cdr(fd)

	} else {
		path = fmt.Sprintf("%s/cdr.%s", os.Getenv("CDR_DIR"), w.TimeStamp)

		fd, err := os.OpenFile(path,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)

		if err != nil {
			loggers.ErrorLogger().Major("File Open Fail")
			return err

		}

		defer fd.Close()
		w.Record_Cdr(fd)

	}
	return nil

}

func (w *CdrMgr) DifferDataCheck() bool {
	now := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d",
		now.Year(), now.Month(), now.Day())

	isOK := strings.Contains(timestamp, w.TimeStamp)
	loggers.InfoLogger().Comment("->>>>>>>>>>>>>>>>>>>>>>Now : %s - FileNow : %s (%v)", timestamp, w.TimeStamp, isOK)

	return isOK

}

func (w *CdrMgr) NonExistFileName() error {
	var path string

	path = fmt.Sprintf("%s/cdr.%s", os.Getenv("CDR_DIR"), w.TimeStamp)
	//preFileName = path

	fd, err := os.OpenFile(path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)

	if err != nil {
		loggers.ErrorLogger().Major("File Open Fail")
		return err

	}

	defer fd.Close()
	w.Record_Cdr(fd)

	return nil
}

func (w *CdrMgr) WriteCdr(
	MsgType string,
	OrigSupi string,
	DestSupi string,
	TimeStamp string,
	DoneTime string,
	Result string,
) error {

	var file os.FileInfo
	var i int
	var Key string

	atomic.AddUint32(&SeqIndex, 1)
	Key = fmt.Sprintf("%d", SeqIndex)

	loggers.InfoLogger().Comment("cdr: key=%s, MsgType=%s, OrigSupi=%s, DestSupi=%s, Result=%s",
		Key, MsgType, OrigSupi, DestSupi, Result)

	w.Key = Key
	w.MsgType = MsgType
	w.OrigSupi = OrigSupi
	w.DestSupi = DestSupi
	w.TimeStamp = TimeStamp
	w.DoneTime = DoneTime
	w.Result = Result

	loggers.InfoLogger().Comment("CDR_DIR : %s", os.Getenv("CDR_DIR"))
	err := os.MkdirAll(os.Getenv("CDR_DIR"), 0755)
	if err != nil {
		loggers.ErrorLogger().Major("Create Fail : Directory")
		return err
	}

	//다시 기동 했을 경우 파일 확인
	check, err := IsEmpty(os.Getenv("CDR_DIR"))
	if err != nil {
		loggers.ErrorLogger().Major("Error : %v", err)
		return err
	}

	if check != false {
		loggers.InfoLogger().Comment("NonExistFile Cdr")
		err = w.NonExistFileName()
		if err != nil {
			loggers.ErrorLogger().Major("Create Fail : CDR")
			return err
		}
		return nil
	}

	fileList, err := ioutil.ReadDir(os.Getenv("CDR_DIR"))
	for i, file = range fileList {
		loggers.InfoLogger().Comment("file-name : %s", file.Name())
	}

	path := fmt.Sprintf("%s/%s", os.Getenv("CDR_DIR"), fileList[i].Name())
	loggers.InfoLogger().Comment("ExistFile Cdr[%s]", path)

	err = w.ExistFileName(path)
	if err != nil {
		loggers.ErrorLogger().Major("Create Fail : CDR")
		return err
	}

	return nil
}

func (w *CdrMgr) Record_Cdr(file *os.File) {

	wr := csv.NewWriter(bufio.NewWriter(file))

	wr.Write([]string{
		w.Key,
		w.MsgType,
		w.OrigSupi,
		w.DestSupi,
		w.TimeStamp,
		w.DoneTime,
		w.Result,
	})
	wr.Flush()
}
