package tcpmgr

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"reflect"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
	jsoniter "github.com/json-iterator/go"
)

var loggers = common.SamsungLoggers()

func (s *TcpServer) MakeMtErrRespSendData(SendData MtData, RespCode int, supi string, Type byte, redis []byte) MtData {
	var data MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.CauseCode = RespCode
	Len = int(reflect.TypeOf(data.CauseCode).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)

	///////////////////////////
	sub := Redis_Response{}
	loggers.InfoLogger().Comment("Redis Get Info : %s", string(redis))
	err := json.Unmarshal(redis, &sub)
	if err != nil {
		loggers.InfoLogger().Comment("Mt Response Redis info Unmarshal Fail (supi): %s", supi)
		return data
	}

	data.Diag_id_len = len(sub.Diag_id)
	copy(data.Diag_id[:data.Diag_id_len], sub.Diag_id)
	Len = Len + data.Diag_id_len + int(reflect.TypeOf(data.Diag_id_len).Size())

	data.Acn = sub.Acn
	Len = Len + int(reflect.TypeOf(data.Acn).Size())

	data.Prov_id = sub.Prov_id
	Len = Len + int(reflect.TypeOf(data.Prov_id).Size())

	data.Inv_id = sub.Inv_id
	Len = Len + int(reflect.TypeOf(data.Inv_id).Size())

	data.Hop_id = sub.Hop_id
	Len = Len + int(reflect.TypeOf(data.Hop_id).Size())

	data.End_id = sub.End_id
	Len = Len + int(reflect.TypeOf(data.End_id).Size())

	data.Peer_id = sub.Peer_id
	Len = Len + int(reflect.TypeOf(data.Peer_id).Size())

	data.Orig_realm_len = len(sub.Orig_realm)
	copy(data.Orig_realm[:data.Orig_realm_len], sub.Orig_realm)
	Len = Len + data.Orig_realm_len + int(reflect.TypeOf(data.Orig_realm_len).Size())

	data.Orig_host_len = len(sub.Orig_host)
	copy(data.Orig_host[:data.Orig_host_len], sub.Orig_host)
	Len = Len + data.Orig_host_len + int(reflect.TypeOf(data.Orig_host_len).Size())

	data.Smsc_node_len = len(sub.Smsc_node)
	copy(data.Smsc_node[:data.Smsc_node_len], sub.Smsc_node)
	Len = Len + data.Smsc_node_len + int(reflect.TypeOf(data.Smsc_node_len).Size())

	data.Session_id_len = len(sub.Session_id)
	copy(data.Session_id[:data.Session_id_len], sub.Session_id)
	Len = Len + data.Session_id_len + int(reflect.TypeOf(data.Session_id_len).Size())

	/////////////////////////////

	data.ContentDataLen = SendData.ContentDataLen
	copy(data.ContentData[:data.ContentDataLen], SendData.ContentData[:data.ContentDataLen])
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())

	data.TotalLen = Len

	return data
}

func (s *TcpServer) MakeMtErrRespSendDataForDiameter(SendData MtData, RespCode int, supi string, Type byte) MtData {
	var data MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.CauseCode = RespCode
	Len = int(reflect.TypeOf(data.CauseCode).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)

	data.ContentDataLen = SendData.ContentDataLen
	copy(data.ContentData[:data.ContentDataLen], SendData.ContentData[:data.ContentDataLen])
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())

	data.TotalLen = Len

	return data
}

func MakeHttpIfMsgResp(SendData MtData) (reqBody []byte, err error) {
	//	var b bytes.Buffer
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var rpdata []byte

	contentsId := fmt.Sprintf("%s@smsf.com", msg5g.RandASCIIBytes(10))
	gpsi := string(SendData.Gpsi[:SendData.GpsiLen])

	rpdata = SendData.ContentData[:SendData.ContentDataLen]
	loggers.InfoLogger().Comment("Send Data(mo-resp) : %d", rpdata)

	request := HttpIfMoMsg{
		ContetnsId: contentsId,
		Gpsi:       gpsi,
		RpData:     rpdata,
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		loggers.ErrorLogger().Major("Json Marshal Fail : %s", err.Error())
		return reqBody, err
	}

	return reqBody, err

}

func MakeHttpIfMsgMt(SendData MtData) ([]byte, error) {
	//var b bytes.Buffer
	var mms bool

	contentsId := fmt.Sprintf("%s@smsf.com", msg5g.RandASCIIBytes(10))
	if SendData.Mms == 1 {
		mms = true
	} else {
		mms = false
	}

	rpdata := SendData.ContentData[:SendData.ContentDataLen]
	request := HttpIfMtMsg{
		ContetnsId: contentsId,
		Mms:        mms,
		RpData:     rpdata,
	}

	reqBody, err := json.Marshal(request)

	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		return reqBody, err
	}

	return reqBody, err
	/*
		w := related.NewWriter(&b)
		w.SetBoundary("Boundary")

		rootPart, err := w.CreateRoot("", "application/json", nil)
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			return nil
		}

		rootPart.Write(reqBody)
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", "application/vnd.3gpp.sms")

		nextPart, err := w.CreatePart(contentsId, header)
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			return nil
		}

		var rpdata []byte
		rpdata = SendData.ContentData[:SendData.ContentDataLen]
		nextPart.Write(rpdata)

		if err := w.Close(); err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			return nil

		}
		loggers.InfoLogger().Comment("Body : %s", b.String())

		return b.Bytes()
	*/
}

func ConvertSendToSvcPodMsg(data []byte, DataLen int) (ret MtData) { // 여기 수정해야 할 부분
	var offset int
	offset = 0

	loggers.InfoLogger().Comment("Recv MSG From MsgProxy(DataLen : %d", DataLen)
	Tmp := make([]byte, 4)

	copy(Tmp[:], data[offset:offset+3])
	ret.Type = byte(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4
	//	loggers.ErrorLogger().Major("offset(%d), type(%d)", offset, ret.Type)
	if ret.Type == MO_RESP {
		copy(Tmp[:], data[offset:offset+3])
		ret.Result = byte(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4
	}

	copy(Tmp[:], data[offset:offset+3])
	ret.CauseCode = int(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4
	//	loggers.ErrorLogger().Major("offset(%d), CauseCode(%d)", offset, ret.CauseCode)

	copy(Tmp[:], data[offset:offset+3])
	ret.SupiLen = int(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4
	//	loggers.ErrorLogger().Major("offset(%d), SupiLen(%d)", offset, ret.SupiLen)

	copy(ret.Supi[:ret.SupiLen], data[offset:offset+ret.SupiLen])
	//	logger.Debug("Supi : %s", string(ret.Supi[:ret.SupiLen]))
	offset = offset + 128

	if ret.Type == MO_RESP {

		copy(Tmp[:], data[offset:offset+3])
		ret.GpsiLen = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Gpsi[:ret.GpsiLen], data[offset:offset+ret.GpsiLen])
		offset = offset + 128
	}

	copy(Tmp[:], data[offset:offset+3])
	ret.MmsLen = byte(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4

	copy(Tmp[:], data[offset:offset+3])
	ret.Mms = byte(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4

	/////////////////////////////////////////

	if ret.Type == MT_MSG {

		copy(Tmp[:], data[offset:offset+3])
		ret.Diag_id_len = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Diag_id[:ret.Diag_id_len], data[offset:offset+ret.Diag_id_len])
		offset = offset + 32

		copy(Tmp[:], data[offset:offset+3])
		ret.Acn = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.Prov_id = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.Inv_id = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.Hop_id = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.End_id = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.Peer_id = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(Tmp[:], data[offset:offset+3])
		ret.Orig_realm_len = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Orig_realm[:ret.Orig_realm_len], data[offset:offset+ret.Orig_realm_len])
		offset = offset + 24

		copy(Tmp[:], data[offset:offset+3])
		ret.Orig_host_len = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Orig_host[:ret.Orig_host_len], data[offset:offset+ret.Orig_host_len])
		offset = offset + 24

		copy(Tmp[:], data[offset:offset+3])
		ret.Smsc_node_len = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Smsc_node[:ret.Smsc_node_len], data[offset:offset+ret.Smsc_node_len])
		offset = offset + 24

		copy(Tmp[:], data[offset:offset+3])
		ret.Session_id_len = int(binary.LittleEndian.Uint32(Tmp))
		offset = offset + 4

		copy(ret.Session_id[:ret.Session_id_len], data[offset:offset+ret.Session_id_len])
		offset = offset + 512

	}

	copy(Tmp[:], data[offset:offset+3])
	ret.ContentDataLen = int(binary.LittleEndian.Uint32(Tmp))
	offset = offset + 4

	if ret.ContentDataLen > 0 && ret.ContentDataLen < 251 {
		copy(ret.ContentData[:ret.ContentDataLen], data[offset:offset+ret.ContentDataLen])
		offset = offset + ret.ContentDataLen
	}

	return ret
}

func (s *TcpServer) InsRedisMtMsg(msg MtData) (val bool) {

	supi := string(msg.Supi[:msg.SupiLen])
	redis := Redis_Response{}

	redis.Acn = msg.Acn
	redis.Diag_id = string(msg.Diag_id[:msg.Diag_id_len])
	redis.End_id = msg.End_id
	redis.Hop_id = msg.Hop_id
	redis.Inv_id = msg.Inv_id
	redis.Orig_host = string(msg.Orig_host[:msg.Orig_host_len])
	redis.Orig_realm = string(msg.Orig_realm[:msg.Orig_realm_len])
	redis.Peer_id = msg.Peer_id
	redis.Prov_id = msg.Prov_id
	redis.Session_id = string(msg.Session_id[:msg.Session_id_len])
	redis.Smsc_node = string(msg.Smsc_node[:msg.Smsc_node_len])

	input, err := json.Marshal(redis)
	if err != nil {
		loggers.InfoLogger().Comment("Json Marshal Fail For insData")
		return false
	}

	loggers.InfoLogger().Comment("Resp Data Info in Redis : %s", string(input))

	rval, _ := s.redisDao.GetSubBySUPI(supi)
	if rval == 1 {
		rval := s.redisDao.InsSub(supi, input)
		if rval == -1 {
			loggers.ErrorLogger().Major("Redis DB Insert Fail")
			return false
		}

	} else {
		loggers.ErrorLogger().Major("Does Not Find Subs Info, supi : %s", supi)
	}

	rval = s.redisDao.InsSub(supi, input)
	if rval == -1 {
		loggers.ErrorLogger().Major("Redis DB Insert Fail, supi : %s", supi)
		return false
	}

	return true

}
