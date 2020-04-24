package controller

import (
	"bytes"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	cpdu "camel.uangel.com/ua5g/usmsf.git/endecoder/cpdu"
)

func MakeCPAck(supi string, cpdata *cpdu.Cpmessage) []byte {

	var cpMsg cpdu.CpEncode

	ulog.Info("MakeCPAck(), USER : %s ", supi)

	cpMsg.Direction = cpdu.DIRECTION_N_MS
	cpMsg.ProtocolDiscr = 9

	TransactionId := (((cpdata.TransactionId & 0x70) | 0x80) >> 4)
	cpMsg.TransactionId = TransactionId
	cpMsg.MessageType = cpdu.TypeCpAck

	encodedData := cpdu.EncodingAck(cpMsg)

	ulog.Debug("CP ACK : %X", encodedData)

	return encodedData
}

func MakeCPErr(cc int, cpdata *cpdu.Cpmessage) []byte {

	var cpMsg cpdu.CpEncode

	cpMsg.Direction = cpdu.DIRECTION_N_MS
	cpMsg.ProtocolDiscr = 9

	TransactionId := (((cpdata.TransactionId & 0x70) | 0x80) >> 4)
	cpMsg.TransactionId = TransactionId
	cpMsg.MessageType = cpdu.TypeCpError

	cpMsg.CpError = byte(cc)
	encodedData := cpdu.EncodingAck(cpMsg)

	ulog.Debug("CP ERROR : %X", encodedData)

	return encodedData
}

func MakeCPData(rpdata []byte) []byte {

	var cpMsg cpdu.CpEncode

	cpMsg.Direction = cpdu.DIRECTION_N_MS
	cpMsg.ProtocolDiscr = 9

	cpMsg.TransactionId = cpdu.CP_TI_VALUE_0_0

	cpMsg.MessageType = cpdu.TypeCpData
	len := bytes.NewReader(rpdata).Len()
	cpMsg.LengthInd = byte(len)

	copy(cpMsg.CpData[0:248], rpdata)

	encodedData := cpdu.EncodingData(cpMsg)

	ulog.Debug("CP DATA : %X", encodedData)

	return encodedData
}
