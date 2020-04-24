package pdu

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
)

/********************************************************
define seo test code
*********************************************************/

func (msg *Cpmessage) CpEncode() (ret []byte) {

	var duHead []byte
	var RpHead []byte
	var i int

	if msg.MessageType == TypeCpData && msg.CpError == CP_SUCC {
		// CP encode
		duHead = msg.MakeCpDuHead()
		if msg.CpError != CP_SUCC {
			ulog.Error("Fail : the reason is makecpduhead making fail")
			return
		}

		for i = range duHead {
			ret = append(ret, duHead[i])

		}

		CpDataLen := msg.LengthInd
		ret = append(ret, CpDataLen)

		RpHead = msg.CpUserData
		if msg.CpError != CP_SUCC {
			ulog.Info("Fail : MakeRpAddr() -Addr Result")
			return
		}

		for i = range RpHead {
			ret = append(ret, RpHead[i])
		}
	} else {
		ulog.Error("Invalid MessageType")
	}

	return
}

func (msg *Cpmessage) CpEncodeAck() (ret []byte) {
	var CpCause byte
	var duHead []byte
	var i int

	// CP encode
	duHead = msg.MakeCpDuHead()

	if msg.MessageType == TypeCpAck && msg.CpError == CP_SUCC {

		for i = range duHead {
			ret = append(ret, duHead[i])
		}

	} else if msg.MessageType == TypeCpError && msg.CpError != CP_SUCC {

		CpCause = msg.CpError
		for i = range duHead {
			ret = append(ret, duHead[i])
		}
		ret = append(ret, CpCause)

	} else {
		ulog.Error("Invalid MessageType")
	}

	msg.CpError = CP_SUCC //Decoding SUCC
	return
}

// Header information, part one
func (msg *Cpmessage) MakeCpDuHead() (ret []byte) {

	var CpduHead byte

	// CP-TrasactionId
	CpduHead = msg.TransactionId << 4

	//CP-ProtocolDiscr
	CpduHead = CpduHead | (msg.ProtocolDiscr & 0x0F)

	ret = append(ret, CpduHead)

	//CpMessasgeType
	ret = append(ret, msg.MessageType)

	return
}
