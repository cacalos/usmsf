package pdu

import (
	"camel.uangel.com/ua5g/usmsf.git/common"
)

var loggers = common.SamsungLoggers()

func Decoding(Input []byte) (ret *Cpmessage) {

	var m *Cpmessage

	m = new(Cpmessage)

	m.CpScan(Input)

	ret = m

	return
}

func EncodingData(CpData CpEncode) (ret []byte) {
	var m *Cpmessage

	if CpData.LengthInd == 0 {
		loggers.ErrorLogger().Major("Mandatory fields missing")
		return
	}

	m = new(Cpmessage)

	// Set Value
	if CpData.CpError == CP_SUCC {
		m.MessageType = TypeCpData
		loggers.InfoLogger().Comment("CpEncode MessageType: %d", m.MessageType)
	} else {
		loggers.ErrorLogger().Major("CpEncoder Func return(cc : %d", CpData.CpError)
		return
	}

	m.Dir = CpData.Direction
	m.ProtocolDiscr = CpData.ProtocolDiscr
	m.TransactionId = CpData.TransactionId

	m.LengthInd = CpData.LengthInd
	m.CpUserData = CpData.CpData[:CpData.LengthInd]

	m.CpError = CpData.CpError

	// Message encode
	ret = m.CpEncode()

	return
}

func EncodingAck(CpData CpEncode) (ret []byte) {
	var m *Cpmessage

	m = new(Cpmessage)

	// Set Value
	m.Dir = CpData.Direction
	m.ProtocolDiscr = CpData.ProtocolDiscr
	m.TransactionId = CpData.TransactionId

	if CpData.CpError == CP_SUCC {
		m.MessageType = TypeCpAck
		loggers.InfoLogger().Comment("MessageType is CpAck(%d)", m.MessageType)
	} else {
		m.MessageType = TypeCpError
		loggers.InfoLogger().Comment("MessageType is CpError(%d)", m.MessageType)
	}

	m.LengthInd = CpData.LengthInd
	m.CpUserData = CpData.CpData[:CpData.LengthInd]

	m.CpError = CpData.CpError

	// Message encode
	ret = m.CpEncodeAck()

	return

}
