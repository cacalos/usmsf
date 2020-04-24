package pdu

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
)

/****************************************************
Insert seo test code
*****************************************************/
// CpEncoder SMS encoder
func (pdu *cpimpl) CpEncoder(inp CpEncode) (ret []byte, err byte) {
	var m *Cpmessage

	if inp.LengthInd == 0 {
		ulog.Error("Not Exist Mandatory filed ")
		return
	}

	m = new(Cpmessage)

	// Set Value
	if inp.CpError == CP_SUCC {
		m.MessageType = TypeCpData
		ulog.Info("CpEncode MessageType : %d", m.MessageType)
	} else {
		ulog.Error("CpEncoder Func return(cc : %d", inp.CpError)
		return
	}

	m.Dir = inp.Direction
	m.ProtocolDiscr = inp.ProtocolDiscr
	m.TransactionId = inp.TransactionId

	m.LengthInd = inp.LengthInd
	m.CpUserData = inp.CpData[:inp.LengthInd]

	m.CpError = inp.CpError

	// Message encode
	ret = m.CpEncode()
	err = m.CpError

	return
}

func (pdu *cpimpl) CpEncoderAck(inp CpEncode) (ret []byte, err byte) {

	var m *Cpmessage

	m = new(Cpmessage)

	// Set Value
	m.Dir = inp.Direction
	m.ProtocolDiscr = inp.ProtocolDiscr
	m.TransactionId = inp.TransactionId

	if inp.CpError == CP_SUCC {
		m.MessageType = TypeCpAck
		ulog.Info("MessageType is CpAck(%d)", m.MessageType)
	} else {
		m.MessageType = TypeCpError
		ulog.Info("MessageType is CpError(%d)", m.MessageType)
	}

	m.LengthInd = inp.LengthInd
	m.CpUserData = inp.CpData[:inp.LengthInd]

	m.CpError = inp.CpError

	// Message encode
	ret = m.CpEncodeAck()
	err = m.CpError

	return

}
