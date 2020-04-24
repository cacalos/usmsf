package msg5g

import (
	//"bytes"
	"fmt"
	//"net/textproto"
	//"sync"

	jsoniter "github.com/json-iterator/go"
	//"github.com/philippfranke/multipart-related/related"

	"camel.uangel.com/ua5g/ulib.git/ulog"
)

type SmsRequest struct {
	ContentsId string `json:"contentsId,omitempty"`
	Gpsi       string `json:"gpsi,omitempty"`
	MmsFlag    bool   `json:"mms,omitempty"`
}

type SmsResp struct {
	ContentsId string `json:"contentsId,omitempty"`
	MsgType    string `json:"msgType,omitempty"`
	ResultCode string `json:"resultCode, omitempty"`
	Rpmsg      []byte `json:"Rpmsg,omitempty"`
	/* For CDR*/
	Result byte `json:"Result,omitempty"`
}

type MoSMS struct {
	Rpmsg      []byte `json:"Rpmsg,omitempty"`
	Gpsi       string `json:"Gpsi,omitempty"`
	ContentsId string `json:"contentsId,omitempty"`
	//	Boundary   string `json:"Boundary,omitempty"`
}

type MtResp struct {
	Rpmsg  []byte `json:"Rpmsg,omitempty"`
	Result byte   `json:"Result,omitempty"`
	//	MsgType    string `json:"MsgType,omitempty"`
	//	ResultCode string `json:"ResultCode,omitempty"`
}

type MtFailNoti struct {
	MsgType    string
	ResultCode string
}

func (m MoSMS) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	contentsId := fmt.Sprintf("%s@smsf.com", RandASCIIBytes(10))

	//	mutex.Lock()
	request := MoSMS{
		ContentsId: contentsId,
		Gpsi:       m.Gpsi,
		Rpmsg:      m.Rpmsg,
	}

	reqBody, err = json.Marshal(request)
	//	mutex.Unlock()

	if err != nil {
		ulog.Info("json.Marshal Error, [%s]", err)
		return reqBody, err
	}
	return reqBody, err

}

func (m MtResp) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//var mutex = &sync.Mutex{}

	contentsId := fmt.Sprintf("%s@smsf.com", RandASCIIBytes(10))

	//	mutex.Lock()
	request := SmsResp{
		ContentsId: contentsId,
		Rpmsg:      m.Rpmsg,
		Result:     m.Result,
	}

	reqBody, err = json.Marshal(request)
	//	mutex.Unlock()

	if err != nil {
		ulog.Info("json.Marshal Error, [%s]", err)
		return reqBody, err
	}
	return reqBody, err
}

func (m MtFailNoti) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	//	mutex.Lock()
	request := SmsResp{
		MsgType:    m.MsgType,
		ResultCode: m.ResultCode,
	}

	reqBody, err = json.Marshal(request)
	//	mutex.Unlock()

	if err != nil {
		ulog.Info("json.Marshal Error, [%s]", err)
		return reqBody, err
	}

	return reqBody, err
}
