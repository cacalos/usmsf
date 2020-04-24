package rpdu

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
)

/****************************************************
Insert seo test code
*****************************************************/
// CpEncoder SMS encoder
func (pdu *rpimpl) RpEncoder(inp RPDU) (ret []byte, err byte) {
	var m *Rpmessage

	m = new(Rpmessage)

	// Set Value
	m.Dir = inp.Direction
	if inp.RpError == RP_SUCC {
		m.RpMessageType = inp.MessageType
	} else {
		ulog.Error("Exist Error Code(%d)", inp.RpError)
		return
	}
	m.RpMessageReference = inp.MessageReference

	// Set originating address number
	if inp.Direction != DIRECTION_MS_N {
		if len(inp.RData.OrigAddr.Min) == 0 {
			//			err = 에러 추가해야함
			ulog.Error("OrigAddr is Empty")
			return
		}

		if inp.RData.OrigAddr.Min[0] == '+' {
			inp.RData.OrigAddr.Min = inp.RData.OrigAddr.Min[1:]
		} else {
			inp.RData.OrigAddr.Min = inp.RData.OrigAddr.Min[0:]
		} //else seo define

		m.RpOrigAddr = inp.RData.OrigAddr.Min
		m.RpOrigTon = inp.RData.OrigAddr.Ton
		m.RpOrigNpi = inp.RData.OrigAddr.Npi

		m.RpOrigAddrLen = uint8(len(m.RpOrigAddr))

	}

	if inp.Direction != DIRECTION_N_MS {
		if len(inp.RData.DestAddr.Min) == 0 {
			ulog.Error("DestAddr is Empty")
			return
		}

		if inp.RData.DestAddr.Min[0] == '+' {
			inp.RData.DestAddr.Min = inp.RData.DestAddr.Min[1:]
		} else {
			inp.RData.DestAddr.Min = inp.RData.DestAddr.Min[0:]
		} //else seo define

		m.RpDestAddr = inp.RData.DestAddr.Min
		m.RpDestTon = inp.RData.DestAddr.Ton
		m.RpDestNpi = inp.RData.DestAddr.Npi

		m.RpDestAddrLen = uint8(len(m.RpDestAddr))

	}

	if len(inp.RData.RpUserData) != 0 {
		m.RpUserDataLen = inp.RData.RpDataLength
		m.RpUserData = inp.RData.RpUserData[:inp.RData.RpDataLength]

	} else {
		ulog.Error("Not exist TPDU that is mandatory field")
		return
	}

	// Message encode
	ret = m.RpEncode()
	err = m.RpError

	return
}

func (pdu *rpimpl) RpEncoderAck(inp RPDU) (ret []byte, err byte) {
	var m *Rpmessage

	m = new(Rpmessage)

	// Set Value
	m.Dir = inp.Direction
	m.RpMessageReference = inp.MessageReference

	if inp.RpError == RP_SUCC {
		ulog.Info("MessageType is RP_ACK_N_MS")
		m.RpMessageType = RP_ACK_N_MS
		m.RpError = inp.RpError

		if inp.RAck.RpDataLength != 0 {
			m.RpLengthInd = inp.RAck.RpDataLength
			m.Tpdu = inp.RAck.RpUserData[:inp.RAck.RpDataLength]
		}

	} else {
		ulog.Info("MessageType is RP_ERROR_N_MS")
		m.RpMessageType = RP_ERROR_N_MS
		m.RpError = inp.RpError

		if inp.RError.RpDataLength != 0 {
			m.RpLengthInd = inp.RError.RpDataLength
			m.Tpdu = inp.RError.RpUserData[:inp.RError.RpDataLength]
		}

	}

	// Message encode
	ret = m.RpEncodeAck()
	err = m.RpError

	return
}

// Set Originating address number
func (pdu *rpimpl) setRpOa(m *Rpmessage, addr string) (err error) {
	if len(addr) == 0 {
		return
	}
	if addr[0] == '+' {
		addr = addr[1:]
	} else {
		addr = addr[0:]
	} //else seo define

	m.RpOrigTon = TON_INTERNATIONAL
	m.RpOrigNpi = NPI_E164

	m.RpOrigAddr = addr

	m.RpOrigAddrLen = uint8(len(m.RpOrigAddr))

	return
}

// Set Destinating address number
func (pdu *rpimpl) setRpDa(m *Rpmessage, addr string) (err error) {
	if len(addr) == 0 {
		return
	}
	if addr[0] == '+' {
		addr = addr[1:]
	} else {
		addr = addr[0:]
	} //else seo define

	m.RpOrigTon = TON_INTERNATIONAL
	m.RpOrigNpi = NPI_E164

	m.RpDestAddr = addr

	m.RpDestAddrLen = uint8(len(m.RpDestAddr))

	return
}
