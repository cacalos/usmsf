package rpdu

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
)

func Decoding(Input []byte) (ret *Rpmessage) {

	var m *Rpmessage

	m = new(Rpmessage)

	m.Dir = DIRECTION_N_MS

	m.CpScan(Input)

	ret = m
	return
}

func EncodingData(RpData RPDU) (ret []byte) {
	var m *Rpmessage

	m = new(Rpmessage)

	// Set Value
	m.Dir = RpData.Direction
	if RpData.RpError == RP_SUCC {
		m.RpMessageType = RpData.MessageType
	} else {
		ulog.Error("Exist Error Code(%d)", RpData.RpError)
		return
	}
	m.RpMessageReference = RpData.MessageReference

	// Set originating address number
	if RpData.Direction != DIRECTION_MS_N {
		if len(RpData.RData.OrigAddr.Min) == 0 {
			//			err = 에러 추가해야함
			ulog.Error("OrigAddr is Empty")
			return
		}

		if RpData.RData.OrigAddr.Min[0] == '+' {
			RpData.RData.OrigAddr.Min = RpData.RData.OrigAddr.Min[1:]
		} else {
			RpData.RData.OrigAddr.Min = RpData.RData.OrigAddr.Min[0:]
		} //else seo define

		m.RpOrigAddr = RpData.RData.OrigAddr.Min
		m.RpOrigTon = RpData.RData.OrigAddr.Ton
		m.RpOrigNpi = RpData.RData.OrigAddr.Npi

		m.RpOrigAddrLen = uint8(len(m.RpOrigAddr))

	}

	if RpData.Direction != DIRECTION_N_MS {
		if len(RpData.RData.DestAddr.Min) == 0 {
			ulog.Error("DestAddr is Empty")
			return
		}

		if RpData.RData.DestAddr.Min[0] == '+' {
			RpData.RData.DestAddr.Min = RpData.RData.DestAddr.Min[1:]
		} else {
			RpData.RData.DestAddr.Min = RpData.RData.DestAddr.Min[0:]
		} //else seo define

		m.RpDestAddr = RpData.RData.DestAddr.Min
		m.RpDestTon = RpData.RData.DestAddr.Ton
		m.RpDestNpi = RpData.RData.DestAddr.Npi

		m.RpDestAddrLen = uint8(len(m.RpDestAddr))

	}

	if len(RpData.RData.RpUserData) != 0 {
		m.RpUserDataLen = RpData.RData.RpDataLength
		m.RpUserData = RpData.RData.RpUserData[:RpData.RData.RpDataLength]

	} else {
		ulog.Error("Not exist TPDU that is mandatory field")
		return
	}

	// Message encode
	ret = m.RpEncode()
	//err = m.RpError

	return
}

func EncodingAck(RpData RPDU) (ret []byte) {
	var m *Rpmessage

	m = new(Rpmessage)

	// Set Value
	m.Dir = RpData.Direction
	m.RpMessageReference = RpData.MessageReference

	if RpData.RpError == RP_SUCC {
		ulog.Info("MessageType is RP_ACK_N_MS")
		m.RpMessageType = RP_ACK_N_MS
		m.RpError = RpData.RpError

		if RpData.RAck.RpDataLength != 0 {
			m.RpLengthInd = RpData.RAck.RpDataLength
			m.Tpdu = RpData.RAck.RpUserData[:RpData.RAck.RpDataLength]
		}

	} else {
		ulog.Info("MessageType is RP_ERROR_N_MS")
		m.RpMessageType = RP_ERROR_N_MS
		m.RpError = RpData.RpError

		if RpData.RError.RpDataLength != 0 {
			m.RpLengthInd = RpData.RError.RpDataLength
			m.Tpdu = RpData.RError.RpUserData[:RpData.RError.RpDataLength]
		}

	}

	// Message encode
	ret = m.RpEncodeAck()
	//	err = m.RpError

	return

}
