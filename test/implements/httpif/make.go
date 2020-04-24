package httpif

import (
	"bytes"
	//"encoding/binary"
	"encoding/json"
	"fmt"
	"net/textproto"
	"reflect"

	"camel.uangel.com/ua5g/usmsf.git/implements/configmgr"
	"camel.uangel.com/ua5g/usmsf.git/implements/tcpmgr"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
	"github.com/philippfranke/multipart-related/related"
)

func MakeMoSendData(info *msg5g.MoSMS, supi string, Type byte) (data tcpmgr.MoData, err int) {
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.SupiLen = byte(len(supi))
	copy(data.Supi[:data.SupiLen], supi)
	Len = Len + int(data.SupiLen) + int(reflect.TypeOf(data.SupiLen).Size())

	data.GpsiLen = byte(len(info.Gpsi))
	copy(data.Gpsi[:data.GpsiLen], info.Gpsi)
	Len = Len + int(data.GpsiLen) + int(reflect.TypeOf(data.GpsiLen).Size())

	data.ContentDataLen = byte(len(info.Rpmsg))
	copy(data.ContentData[:data.ContentDataLen], info.Rpmsg)
	Len = Len + int(data.ContentDataLen) + int(reflect.TypeOf(data.ContentDataLen).Size())

	/****************** Config INFO ***********************/
	loggers.InfoLogger().Comment("Config Info(GPSI): %s", info.Gpsi)
	SmscPrefix := configmgr.SmscPrefixStoragePop(info.Gpsi)

	if SmscPrefix == nil {
		loggers.ErrorLogger().Major("SmscPrefixStoragePop() Fail : %s", info.Gpsi)
		return data, tcpmgr.SmscNodeStoragePopFail
	}

	SmscInfo := configmgr.SmscNodeStoragePop(SmscPrefix.SmscName)
	if SmscInfo == nil {
		loggers.ErrorLogger().Major("SmscNodeStoragePop() Fail : %s", SmscPrefix.Prefix)
		return data, tcpmgr.SmscNodeStoragePopFail
	}

	data.InterFInfo.NameLen = len(SmscInfo.Name)
	data.InterFInfo.Name = SmscInfo.Name[:data.InterFInfo.NameLen]
	Len = Len + data.InterFInfo.NameLen + int(reflect.TypeOf(data.InterFInfo.NameLen).Size())

	data.InterFInfo.IsdnLen = len(SmscInfo.Isdn)
	data.InterFInfo.Isdn = SmscInfo.Isdn[:data.InterFInfo.IsdnLen]
	Len = Len + data.InterFInfo.IsdnLen + int(reflect.TypeOf(data.InterFInfo.IsdnLen).Size())

	data.InterFInfo.PcLen = int(reflect.TypeOf(data.InterFInfo.PcLen).Size())
	data.InterFInfo.Pc = SmscInfo.Pc
	Len = Len + int(reflect.TypeOf(data.InterFInfo.Pc).Size()) + int(reflect.TypeOf(data.InterFInfo.PcLen).Size())

	data.InterFInfo.SsnLen = int(reflect.TypeOf(data.InterFInfo.SsnLen).Size())
	data.InterFInfo.Ssn = SmscInfo.Ssn
	Len = Len + int(reflect.TypeOf(data.InterFInfo.Ssn).Size()) + int(reflect.TypeOf(data.InterFInfo.SsnLen).Size())

	data.InterFInfo.TypeLen = int(reflect.TypeOf(data.InterFInfo.TypeLen).Size())
	data.InterFInfo.Type = SmscInfo.Type
	Len = Len + int(reflect.TypeOf(data.InterFInfo.Type).Size()) + int(reflect.TypeOf(data.InterFInfo.TypeLen).Size())

	data.InterFInfo.FlowCtrlLen = int(reflect.TypeOf(data.InterFInfo.FlowCtrlLen).Size())
	data.InterFInfo.FlowCtrl = SmscInfo.FlowCtrl
	Len = Len + int(reflect.TypeOf(data.InterFInfo.FlowCtrl).Size()) + int(reflect.TypeOf(data.InterFInfo.FlowCtrlLen).Size())

	data.InterFInfo.DestHostLen = len(SmscInfo.Dest_host)
	data.InterFInfo.DestHost = SmscInfo.Dest_host[:data.InterFInfo.DestHostLen]
	Len = Len + data.InterFInfo.DestHostLen + int(reflect.TypeOf(data.InterFInfo.DestHostLen).Size())

	data.InterFInfo.DestRealmLen = len(SmscInfo.Dest_realm)
	data.InterFInfo.DestRealm = SmscInfo.Dest_realm[:data.InterFInfo.DestRealmLen]
	Len = Len + data.InterFInfo.DestRealmLen + int(reflect.TypeOf(data.InterFInfo.DestRealmLen).Size())

	data.InterFInfo.DescLen = len(SmscInfo.Desc)
	data.InterFInfo.Desc = SmscInfo.Desc[:data.InterFInfo.DescLen]
	Len = Len + data.InterFInfo.DescLen + int(reflect.TypeOf(data.InterFInfo.DescLen).Size())

	///////////////////////////////////////////////
	Common := configmgr.CommonStoragePop()

	if Common == nil {
		loggers.ErrorLogger().Major("Parse Common Config Error")
		err = tcpmgr.CommonStoragePopFail
		return data, err
	}

	data.CommonConf.PlmnIdLen = int(reflect.TypeOf(data.CommonConf.PlmnIdLen).Size())
	data.CommonConf.PlmnId = Common.PlmnId
	Len = Len + int(reflect.TypeOf(data.CommonConf.PlmnId).Size()) + int(reflect.TypeOf(data.CommonConf.PlmnIdLen).Size())

	data.CommonConf.SmsfInstanceIdLen = len(Common.SmsfInstanceId)
	data.CommonConf.SmsfInstanceId = Common.SmsfInstanceId[:data.CommonConf.SmsfInstanceIdLen]
	Len = Len + data.CommonConf.SmsfInstanceIdLen + int(reflect.TypeOf(data.CommonConf.SmsfInstanceIdLen).Size())

	data.CommonConf.SmsfMapAddressLen = len(Common.SmsfMapAddress)
	data.CommonConf.SmsfMapAddress = Common.SmsfMapAddress[:data.CommonConf.SmsfMapAddressLen]
	Len = Len + data.CommonConf.SmsfMapAddressLen + int(reflect.TypeOf(data.CommonConf.SmsfMapAddressLen).Size())

	data.CommonConf.SmsfDiameterAddressLen = len(Common.SmsfDiameterAddress)
	data.CommonConf.SmsfDiameterAddress = Common.SmsfDiameterAddress[:data.CommonConf.SmsfDiameterAddressLen]
	Len = Len + data.CommonConf.SmsfDiameterAddressLen + int(reflect.TypeOf(data.CommonConf.SmsfDiameterAddressLen).Size())

	data.CommonConf.SmsfPointCodeLen = int(reflect.TypeOf(data.CommonConf.SmsfPointCodeLen).Size())
	data.CommonConf.SmsfPointCode = Common.SmsfPointCode
	Len = Len + int(reflect.TypeOf(data.CommonConf.SmsfPointCode).Size()) + int(reflect.TypeOf(data.CommonConf.SmsfPointCodeLen).Size())

	data.CommonConf.SmsfSsnLen = int(reflect.TypeOf(data.CommonConf.SmsfSsnLen).Size())
	data.CommonConf.SmsfSsn = Common.SmsfSsn
	Len = Len + int(reflect.TypeOf(data.CommonConf.SmsfSsn).Size()) + int(reflect.TypeOf(data.CommonConf.SmsfSsnLen).Size())
	data.TotalLen = Len

	return data, 0

}

//////
func (s *IfServer) MakeMtRespNotiSendData(info *msg5g.SmsResp, BinData []byte, supi string, Type byte, redis []byte) tcpmgr.MtData {
	var data tcpmgr.MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)
	Len = Len + int(data.SupiLen) + int(reflect.TypeOf(data.SupiLen).Size())

	data.MsgTypeLen = len(info.MsgType)
	copy(data.MsgType[:data.MsgTypeLen], info.MsgType)
	Len = Len + int(data.MsgTypeLen) + int(reflect.TypeOf(data.MsgTypeLen).Size())

	data.ResultCodeLen = len(info.ResultCode)
	copy(data.ResultCode[:data.ResultCodeLen], info.ResultCode)
	Len = Len + int(data.ResultCodeLen) + int(reflect.TypeOf(data.ResultCodeLen).Size())
	/*
		if info.Mms == true {
			data.MmsLen = 1
			data.Mms = 1
		} else {
			data.MmsLen = 1
			data.Mms = 0
		}
		Len = Len + int(reflect.TypeOf(data.MmsLen).Size()) + int(reflect.TypeOf(data.Mms).Size()) //Mms Len + MMS flag
	*/
	sub := tcpmgr.Redis_Response{}
	err := json.Unmarshal(redis, &sub)
	if err != nil {
		loggers.ErrorLogger().Major("Mt Response Redis info Unmarshal Fail (supi): %s", supi)
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

	data.ContentDataLen = len(BinData)
	copy(data.ContentData[:data.ContentDataLen], BinData)
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())
	data.TotalLen = Len

	return data
}

//////
func (s *IfServer) MakeMtRespNotiSendDataForDiameter(info *msg5g.SmsResp, BinData []byte, supi string, Type byte) tcpmgr.MtData {
	var data tcpmgr.MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)
	Len = Len + int(data.SupiLen) + int(reflect.TypeOf(data.SupiLen).Size())

	data.MsgTypeLen = len(info.MsgType)
	copy(data.MsgType[:data.MsgTypeLen], info.MsgType)
	Len = Len + int(data.MsgTypeLen) + int(reflect.TypeOf(data.MsgTypeLen).Size())

	data.ResultCodeLen = len(info.ResultCode)
	copy(data.ResultCode[:data.ResultCodeLen], info.ResultCode)
	Len = Len + int(data.ResultCodeLen) + int(reflect.TypeOf(data.ResultCodeLen).Size())
	data.ContentDataLen = len(BinData)
	copy(data.ContentData[:data.ContentDataLen], BinData)
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())
	data.TotalLen = Len

	return data
}

func (s *IfServer) MakeMtRespSendDataForDiameter(info *msg5g.SmsResp, BinData []byte, supi string, Type byte) tcpmgr.MtData {
	var data tcpmgr.MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)
	Len = Len + int(data.SupiLen) + int(reflect.TypeOf(data.SupiLen).Size())
	data.ContentDataLen = len(BinData)
	copy(data.ContentData[:data.ContentDataLen], BinData)
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())

	data.TotalLen = Len

	return data
}

func (s *IfServer) MakeMtRespSendData(info *msg5g.SmsResp, BinData []byte, supi string, Type byte, redis []byte) tcpmgr.MtData {
	var data tcpmgr.MtData
	var Len int

	data.Type = Type
	Len = int(reflect.TypeOf(data.Type).Size())

	data.SupiLen = len(supi)
	copy(data.Supi[:data.SupiLen], supi)
	Len = Len + int(data.SupiLen) + int(reflect.TypeOf(data.SupiLen).Size())

	//data.Result =

	sub := tcpmgr.Redis_Response{}
	err := json.Unmarshal(redis, &sub)
	if err != nil {
		loggers.ErrorLogger().Major("Mt Response Redis info Unmarshal Fail (supi): %s", supi)
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

	data.ContentDataLen = len(BinData)
	copy(data.ContentData[:data.ContentDataLen], BinData)
	Len = Len + data.ContentDataLen + int(reflect.TypeOf(data.ContentDataLen).Size())

	data.TotalLen = Len

	return data
}

func MakeHttpIfMsgResp(SendData tcpmgr.MtData) []byte {
	var b bytes.Buffer
	contentsId := fmt.Sprintf("%s@smsf.com", msg5g.RandASCIIBytes(10))
	gpsi := string(SendData.Gpsi[:SendData.GpsiLen])

	request := tcpmgr.HttpIfMoMsg{
		ContetnsId: contentsId,
		Gpsi:       gpsi,
	}

	reqBody, err := json.Marshal(request)

	if err != nil {
		loggers.ErrorLogger().Major("json.Marshal Fail : %s", err.Error())
		return nil
	}

	w := related.NewWriter(&b)
	w.SetBoundary("Boundary")

	rootPart, err := w.CreateRoot("", "application/json", nil)
	if err != nil {
		loggers.ErrorLogger().Major("CreateRoot Part Fail : %s", err.Error())
		return nil
	}

	rootPart.Write(reqBody)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/vnd.3gpp.sms")

	nextPart, err := w.CreatePart(contentsId, header)
	if err != nil {
		loggers.ErrorLogger().Major("Create Part Fail : %s", err.Error())
		return nil
	}

	var cpdata []byte
	cpdata = SendData.ContentData[:SendData.ContentDataLen]
	nextPart.Write(cpdata)

	if err := w.Close(); err != nil {
		loggers.ErrorLogger().Major("Make Response Message Close Fail: %s", err.Error())
		return nil
	}
	loggers.InfoLogger().Comment("Create Body %s", b.String())

	return b.Bytes()

}
