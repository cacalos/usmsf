package httpif

import (
	"encoding/binary"
	"reflect"

	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/tcpmgr"
)

var loggers = common.SamsungLoggers()

func (s *IfServer) SendRespToMsgProxyForDiameter(input tcpmgr.MtData) error {
	var err error
	errchan := make(chan error)

	loggers.InfoLogger().Comment("Send Resp msg To msgProxy")
	//	defer client.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음
	exec.SafeGo(func() {

		var offset int
		var i int
		Len := make([]byte, 4)

		binary.LittleEndian.PutUint32(Len, uint32(input.TotalLen+4))

		buf := make([]byte, input.TotalLen+4)

		copy(buf[0:3], Len[0:3])
		offset = offset + 4

		buf[offset] = input.Type
		offset++

		buf[offset] = byte(input.SupiLen)
		offset++
		for i = 0; i < int(input.SupiLen); i++ {
			buf[offset+i] = input.Supi[i]
		}
		offset = offset + i

		buf[offset] = input.MmsLen
		offset++

		buf[offset] = input.Mms
		offset++

		buf[offset] = byte(input.MsgTypeLen)
		offset++
		copy(buf[offset:], input.MsgType[:input.MsgTypeLen])
		offset = offset + input.MsgTypeLen

		buf[offset] = byte(input.ResultCodeLen)
		offset++
		copy(buf[offset:], input.ResultCode[:input.ResultCodeLen])
		offset = offset + input.ResultCodeLen

		buf[offset] = byte(input.ContentDataLen)
		offset++
		copy(buf[offset:], input.ContentData[:input.ContentDataLen])
		offset = offset + int(input.ContentDataLen)

		_, err = s.tcpInfo.Client.Write(buf) // 서버로 데이터를 보냄
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			errchan <- err
			return
		}

	})

	select {
	case err1 := <-errchan:
		if err1 != nil {
			close(errchan)
			return err1
		}
	default:

		close(errchan)

	}

	//	close(errchan)
	return nil
}

func (s *IfServer) SendRespToMsgProxy(input tcpmgr.MtData) error {
	var err error
	errchan := make(chan error)

	loggers.InfoLogger().Comment("Send Resp msg To msgProxy")
	//	defer client.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음
	exec.SafeGo(func() {

		var offset int
		var i int
		Len := make([]byte, 4)

		binary.LittleEndian.PutUint32(Len, uint32(input.TotalLen+4))

		buf := make([]byte, input.TotalLen+4)

		copy(buf[0:3], Len[0:3])
		offset = offset + 4

		buf[offset] = input.Type
		offset++

		buf[offset] = byte(input.SupiLen)
		offset++
		for i = 0; i < int(input.SupiLen); i++ {
			buf[offset+i] = input.Supi[i]
		}
		offset = offset + i

		buf[offset] = input.MmsLen
		offset++

		buf[offset] = input.Mms
		offset++

		buf[offset] = byte(input.MsgTypeLen)
		offset++
		copy(buf[offset:], input.MsgType[:input.MsgTypeLen])
		offset = offset + input.MsgTypeLen

		buf[offset] = byte(input.ResultCodeLen)
		offset++
		copy(buf[offset:], input.ResultCode[:input.ResultCodeLen])
		offset = offset + input.ResultCodeLen

		buf[offset] = byte(input.Diag_id_len)
		offset++
		copy(buf[offset:], input.Diag_id[:input.Diag_id_len])
		offset = offset + int(input.Diag_id_len)

		binary.LittleEndian.PutUint32(Len, uint32(input.Acn))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		binary.LittleEndian.PutUint32(Len, uint32(input.Prov_id))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		binary.LittleEndian.PutUint32(Len, uint32(input.Inv_id))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		binary.LittleEndian.PutUint32(Len, uint32(input.Hop_id))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		binary.LittleEndian.PutUint32(Len, uint32(input.End_id))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		binary.LittleEndian.PutUint32(Len, uint32(input.Peer_id))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		buf[offset] = byte(input.Orig_realm_len)
		offset++
		copy(buf[offset:], input.Orig_realm[:input.Orig_realm_len])
		offset = offset + int(input.Orig_realm_len)

		buf[offset] = byte(input.Orig_host_len)
		offset++
		copy(buf[offset:], input.Orig_host[:input.Orig_host_len])
		offset = offset + int(input.Orig_host_len)

		buf[offset] = byte(input.Smsc_node_len)
		offset++
		copy(buf[offset:], input.Smsc_node[:input.Smsc_node_len])
		offset = offset + int(input.Smsc_node_len)

		binary.LittleEndian.PutUint32(Len, uint32(input.Session_id_len))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + 4

		copy(buf[offset:], input.Session_id[:input.Session_id_len])
		offset = offset + int(input.Session_id_len)

		buf[offset] = byte(input.ContentDataLen)
		offset++
		copy(buf[offset:], input.ContentData[:input.ContentDataLen])
		offset = offset + int(input.ContentDataLen)

		_, err = s.tcpInfo.Client.Write(buf) // 서버로 데이터를 보냄
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			errchan <- err
			return
		}

	})

	select {
	case err1 := <-errchan:
		if err1 != nil {
			close(errchan)
			return err1
		}
	default:

		close(errchan)

	}

	//	close(errchan)
	return nil
}

func (s *IfServer) SendToMsgProxy(input tcpmgr.MoData) error {

	var err error
	errchan := make(chan error)

	loggers.InfoLogger().Comment("Send msg To msgProxy")

	//	defer client.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음

	exec.SafeGo(func() {
		var offset int
		var i int
		Len := make([]byte, 4)

		binary.LittleEndian.PutUint32(Len, uint32(input.TotalLen+4))

		buf := make([]byte, input.TotalLen+4)

		copy(buf[0:3], Len[0:3])
		offset = offset + 4

		buf[offset] = input.Type
		offset++

		buf[offset] = input.SupiLen
		offset++
		for i = 0; i < int(input.SupiLen); i++ {
			buf[offset+i] = input.Supi[i]
		}
		offset = offset + i

		buf[offset] = input.GpsiLen
		offset++
		for i = 0; i < int(input.GpsiLen); i++ {
			buf[offset+i] = input.Gpsi[i]
		}
		offset = offset + i

		buf[offset] = input.ContentDataLen
		offset++
		copy(buf[offset:], input.ContentData[:input.ContentDataLen])

		offset = offset + int(input.ContentDataLen)

		/****************** Config INFO ***********************/
		// Name
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.NameLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.NameLen).Size())

		for i = 0; i < int(input.InterFInfo.NameLen); i++ {
			buf[offset+i] = input.InterFInfo.Name[i]
		}
		offset = offset + i

		// Isdn
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.IsdnLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.NameLen).Size())

		for i = 0; i < int(input.InterFInfo.IsdnLen); i++ {
			buf[offset+i] = input.InterFInfo.Isdn[i]
		}
		offset = offset + i

		// Pc
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.PcLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.PcLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.Pc))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.Pc).Size())

		//SSN
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.SsnLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.SsnLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.Ssn))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.Ssn).Size())

		//Type
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.TypeLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.TypeLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.Type))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.Type).Size())

		//FlowCont
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.FlowCtrlLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.FlowCtrlLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.FlowCtrl))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.FlowCtrl).Size())

		//Dest_host
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.DestHostLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.DestHostLen).Size())

		for i = 0; i < int(input.InterFInfo.DestHostLen); i++ {
			buf[offset+i] = input.InterFInfo.DestHost[i]
		}
		offset = offset + i

		//Dest_realm
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.DestRealmLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.DestRealmLen).Size())

		for i = 0; i < int(input.InterFInfo.DestRealmLen); i++ {
			buf[offset+i] = input.InterFInfo.DestRealm[i]
		}
		offset = offset + i

		//DESC
		binary.LittleEndian.PutUint32(Len, uint32(input.InterFInfo.DescLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.InterFInfo.DescLen).Size())

		for i = 0; i < int(input.InterFInfo.DescLen); i++ {
			buf[offset+i] = input.InterFInfo.Desc[i]
		}
		offset = offset + i

		//PlmnId
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.PlmnIdLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.PlmnIdLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.PlmnId))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.PlmnId).Size())

		//SmsfInstaceId
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfInstanceIdLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfInstanceIdLen).Size())

		for i = 0; i < int(input.CommonConf.SmsfInstanceIdLen); i++ {
			buf[offset+i] = input.CommonConf.SmsfInstanceId[i]
		}
		offset = offset + i

		//SmsfMapAddr
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfMapAddressLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfMapAddressLen).Size())

		for i = 0; i < int(input.CommonConf.SmsfMapAddressLen); i++ {
			buf[offset+i] = input.CommonConf.SmsfMapAddress[i]
		}
		offset = offset + i

		//SmsfDiaAddr
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfDiameterAddressLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfDiameterAddressLen).Size())

		for i = 0; i < int(input.CommonConf.SmsfDiameterAddressLen); i++ {
			buf[offset+i] = input.CommonConf.SmsfDiameterAddress[i]
		}
		offset = offset + i

		//SmsfPointCode
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfPointCodeLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfPointCodeLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfPointCode))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfPointCode).Size())

		//SmsfSsn
		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfSsnLen))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfSsnLen).Size())

		binary.LittleEndian.PutUint32(Len, uint32(input.CommonConf.SmsfSsn))
		copy(buf[offset:offset+3], Len[0:3])
		offset = offset + int(reflect.TypeOf(input.CommonConf.SmsfSsn).Size())

		_, err = s.tcpInfo.Client.Write(buf) // 서버로 데이터를 보냄
		if err != nil {
			loggers.ErrorLogger().Major("%s", err.Error())
			errchan <- err
			return

		}

	})

	select {
	case err1 := <-errchan:
		if err1 != nil {
			close(errchan)
			return err1
		}
	default:
		close(errchan)
	}

	//close(errchan)
	return nil
}
