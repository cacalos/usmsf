package rpdu

import (
	//"bytes"
	//"encoding/hex"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"fmt"
	"gopkg.in/webnice/pdu.v1/encoders"
)

/*********************************************************
Define seo
***********************************************************/
// Scan Source data scanning
func (msg *Rpmessage) CpScan(src []byte) {

	msg.RpfindPDU(src)

	// Error, PDU data is empty
	if len(msg.DataSource) == 0 {
		ulog.Info("Decoding Source Data is Empty")
		msg.RpError = RP_INFORMATION_ELEMENT_NON_EXISTENT_OR_NOT_IMPLEMENTED // 여기에 에러코드 값 넣어줘야함....
		msg.End = true
		return
	}

	msg.CheckMti(src[0])
	MessageType := msg.RpMessageType

	if MessageType == RP_DATA_MS_N || MessageType == RP_DATA_N_MS {
		//		ulog.Debug("Decoding RPDATA START")
		msg.loadRpDu()
	} else if MessageType == RP_ACK_MS_N || MessageType == RP_ACK_N_MS {
		//		ulog.Debug("Decoding RPACK START")
		msg.loadRpAck()
	} else if MessageType == RP_ERROR_MS_N || MessageType == RP_ERROR_N_MS {
		//		ulog.Debug("Decoding RPERROR START")
		msg.loadRpError()
	} else {
		ulog.Error("Invalid Chkeck MTI")
		msg.RpError = RP_MESSAGE_TYPE_NON_EXISTENT_OR_NOT_IMPLEMENTED
		msg.End = true
		return
	}

	if msg.RpError != RP_SUCC {
		ulog.Error("Decoding Data Fail")
		msg.End = true
		return
	}

	return
}

func (msg *Rpmessage) RpfindPDU(src []byte) {

	msg.DataSource = src
}

func ValidMti(val byte) bool {
	switch val {
	case RP_DATA_MS_N:
		return true
	case RP_DATA_N_MS:
		return true
	case RP_ACK_MS_N:
		return true
	case RP_ACK_N_MS:
		return true
	case RP_ERROR_MS_N:
		return true
	case RP_ERROR_N_MS:
		return true

	}
	return false

}

func (msg *Rpmessage) loadRpError() {
	var tmp byte
	//	var buf []byte
	//	var size uint8
	var flag bool

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageType = tmp
	msg.Lp++

	flag = ValidMti(msg.RpMessageType)
	if flag != true {
		msg.RpError = RP_INVALID_MANDATORY_INFORMATION
		return
	}

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageReference = tmp
	msg.Lp++

	defer func() {
		recover()
	}()

	if msg.DataSource[msg.Lp] != 0 {
		tmp = msg.DataSource[msg.Lp]
		msg.RpLengthInd = tmp
		msg.Lp++ // 이거 TPDU

		msg.RpUserData = msg.DataSource[msg.Lp:]
	}
}

func (msg *Rpmessage) loadRpAck() {
	var tmp byte
	var flag bool

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageType = tmp
	msg.Lp++

	flag = ValidMti(msg.RpMessageType)
	if flag != true {
		msg.RpError = RP_INVALID_MANDATORY_INFORMATION
		fmt.Println("ERROR : Invalid MTI")
		return
	}

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageReference = tmp
	msg.Lp++

	defer func() {
		recover()
	}()

	if msg.DataSource[msg.Lp] != 0 {
		tmp = msg.DataSource[msg.Lp]
		msg.RpLengthInd = tmp
		msg.Lp++ // 이거 TPDU

		msg.RpUserData = msg.DataSource[msg.Lp:]
	} else {
		fmt.Println("Not Exist RP-DATA in RP-ACK")
		return
	}

	fmt.Println("RP-ACK Decoding SUCC.......")
	return
}

func (msg *Rpmessage) CheckMti(mti byte) {
	switch mti {
	case RP_DATA_MS_N:
		msg.RpMessageType = RP_DATA_MS_N
	case RP_DATA_N_MS:
		msg.RpMessageType = RP_DATA_N_MS
	case RP_ACK_MS_N:
		msg.RpMessageType = RP_ACK_MS_N
	case RP_ACK_N_MS:
		msg.RpMessageType = RP_ACK_N_MS
	case RP_ERROR_MS_N:
		msg.RpMessageType = RP_ERROR_MS_N
	case RP_ERROR_N_MS:
		msg.RpMessageType = RP_ERROR_N_MS
	default:
		msg.RpMessageType = RP_INVALID_TYPE
	}
}

func (msg *Rpmessage) loadRpDu() {
	var tmp byte
	var buf []byte
	var size uint8
	var flag bool

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageType = tmp
	msg.Lp++

	flag = ValidMti(msg.RpMessageType)
	if flag != true {
		msg.RpError = RP_INVALID_MANDATORY_INFORMATION
		return
	}

	tmp = msg.DataSource[msg.Lp]
	msg.RpMessageReference = tmp
	msg.Lp++

	//	if msg.Dir == DIRECTION_MS_N {
	if msg.RpMessageType == RP_DATA_MS_N {
		tmp = msg.DataSource[msg.Lp]
		msg.RpOrigAddrLen = tmp
		msg.Lp++
	} else {
		tmp = msg.DataSource[msg.Lp]
		msg.RpOrigAddrLen = tmp
		msg.Lp++

		//ton/npi
		tmp = msg.DataSource[msg.Lp]

		msg.RpOrigNpi = tmp & 0x0F
		msg.RpOrigTon = (tmp >> 4) & 0x07
		msg.Lp++
		//여기 addr. 값 들어가면되는거네..

		size = msg.Lp + msg.RpOrigAddrLen
		size = size - 1

		buf = msg.DataSource[msg.Lp:size]
		msg.RpOrigAddr = encoders.NewSemiOctet().DecodeAddress(buf)
		//		msg.Lp++

		//		msg.RpOrigAddrLen = uint8(len(msg.RpOrigAddr))
		msg.RpOrigAddrLen = uint8(len(buf))

		msg.Lp = msg.Lp + msg.RpOrigAddrLen
		msg.Lp++ // 이거 TPDU

	}

	//	if msg.Dir == DIRECTION_N_MS {
	if msg.RpMessageType == RP_DATA_N_MS {
		tmp = msg.DataSource[msg.Lp]
		msg.RpDestAddrLen = tmp
		msg.Lp++
	} else {

		tmp = msg.DataSource[msg.Lp]
		msg.RpDestAddrLen = tmp
		msg.Lp++

		//ton/npi
		tmp = msg.DataSource[msg.Lp]

		msg.RpDestNpi = tmp & 0x0F
		msg.RpDestTon = (tmp >> 4) & 0x07
		msg.Lp++
		//여기 addr. 값 들어가면되는거네..

		//size = msg.Lp + msg.RpDestAddrLen - 1
		size = msg.Lp + msg.RpDestAddrLen
		size = size - 1
		buf = msg.DataSource[msg.Lp:size]
		msg.RpDestAddr = encoders.NewSemiOctet().DecodeAddress(buf)

		//		msg.RpDestAddrLen = uint8(len(msg.RpDestAddr))
		msg.RpDestAddrLen = uint8(len(buf))

		msg.Lp = msg.Lp + msg.RpDestAddrLen

	}

	if msg.DataSource[msg.Lp] != 0 {
		tmp = msg.DataSource[msg.Lp]
		msg.RpLengthInd = tmp
		msg.Lp++ // 이거 TPDU

		msg.RpUserData = msg.DataSource[msg.Lp:]
	} else {
		msg.RpError = RP_INVALID_MANDATORY_INFORMATION
		return
	}

	return

}
