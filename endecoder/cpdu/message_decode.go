package pdu

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
)

/*********************************************************
Define seo
***********************************************************/
// Scan Source data scanning
func (msg *Cpmessage) CpScan(src []byte) {

	msg.CpfindPDU(src)

	// Error, PDU data is empty
	if len(msg.DataSource) == 0 {
		ulog.Error("Decoding Data is Empty")
		msg.CpError = CP_INVALID_MANDATORY_INFORMATION
		ulog.Info("Set CpError : %d", msg.CpError)
		msg.End = true
		return
	}

	msg.CheckMti(src[1])

	MessageType := msg.MessageType

	msg.loadCpHead()
	if msg.CpError != CP_SUCC {
		ulog.Error("CP Message Decoding Fail -> msg.loadCpHead()")
		ulog.Info("Set CpError : %d", msg.CpError)
		msg.End = true
		return
	}

	if MessageType == TypeCpData {
		msg.loadRpDuHead()
		if msg.CpError != CP_SUCC {
			ulog.Error("CP Message Decoding Fail -> msg.loadRpDuHead()")
			ulog.Info("Set CpError : %d", msg.CpError)
			msg.End = true
			return
		}
	} else if MessageType == TypeCpAck {

	} else if MessageType == TypeCpError {
		//CP-Cause
		msg.CauseValue = src[msg.Lp]

	} else {
		ulog.Error("Invalid CP MessageType")
	}

	return
}

func (msg *Cpmessage) CheckMti(val byte) {

	switch val {
	case TypeCpData:
		msg.MessageType = TypeCpData
	case TypeCpAck:
		msg.MessageType = TypeCpAck
	case TypeCpError:
		msg.MessageType = TypeCpError
	}

}

func (msg *Cpmessage) CpfindPDU(src []byte) {

	msg.DataSource = src
}

func (msg *Cpmessage) loadCpHead() {
	var tmp byte

	msg.Dir = DIRECTION_MS_N

	tmp = msg.DataSource[msg.Lp]

	msg.ProtocolDiscr = tmp & 0x0F

	val := ValidProtocolDeiscr(msg.ProtocolDiscr)
	if val != true {
		msg.CpError = CP_PROTOCOL_ERROR_UNSPECIFIED
		ulog.Error("Error : PROTOCOL_ERROR_UNSPECIFIED(%d)", msg.CpError)
		return
	}

	msg.TransactionId = tmp & 0xF0
	check := msg.TransactionId >> 4
	val = ValidTransactionId(check)

	if val != true {
		msg.CpError = CP_INVALID_TRANSACTION_IDENTIFIER_VALUE
		ulog.Error("Error : INVALID_TRANSACTION_IDENTIFIER_VALUE (%d)", msg.CpError)
		return
	}

	msg.Lp++

	tmp = msg.DataSource[msg.Lp]
	msg.MessageType = tmp

	val = ValidMti(msg.MessageType)

	if val != true {
		msg.CpError = CP_MESSAGE_TYPE_NON_EXISTENT_OR_NOT_IMPLEMENTED
		ulog.Error("Error : MESSAGE_TYPE_NON_EXISTENT_OR_NOT_IMPLEMENTED(%d)", msg.CpError)
		return
	}

	msg.Lp++
}

func ValidMti(val byte) bool {

	switch val {
	case TypeCpData:
		return true
	case TypeCpAck:
		return true
	case TypeCpError:
		return true
	}

	return false

}

func ValidTransactionId(val byte) bool {
	switch val {

	//TI VALUE is 0
	case CP_TI_VALUE_0_0:
		return true
	case CP_TI_VALUE_0_1:
		return true
	case CP_TI_VALUE_0_2:
		return true
	case CP_TI_VALUE_0_3:
		return true
	case CP_TI_VALUE_0_4:
		return true
	case CP_TI_VALUE_0_5:
		return true
	case CP_TI_VALUE_0_6:
		return true
	case CP_TI_TIE_0_VALUE:
		return true

	//TI VALUE is 8
	case CP_TI_VALUE_8_0:
		return true
	case CP_TI_VALUE_8_1:
		return true
	case CP_TI_VALUE_8_2:
		return true
	case CP_TI_VALUE_8_3:
		return true
	case CP_TI_VALUE_8_4:
		return true
	case CP_TI_VALUE_8_5:
		return true
	case CP_TI_VALUE_8_6:
		return true
	case CP_TI_TIE_8_VALUE:
		return true

	}

	return false
}

func ValidProtocolDeiscr(val byte) bool {

	switch val {
	case CP_GORUP_CALL_CONTROL:
		return false
	case CP_BROADCAST_CALL_CONTROL:
		return false
	case CP_EPS_SESSION_MANAGEMENT_MESSAGES:
		return false
	case CP_CALL_CONTROL:
		return false
	case CP_GPRS_TRANSPARENT_TRANSPORT_PROTOCOL:
		return false
	case CP_MOBILITY_MANAGEMENT_MESSAGE:
		return false
	case CP_RADIO_RESOURCES_MANAGEMENT_MESSAGES:
		return false
	case CP_EPS_MOBILITY_MANAGEMENT_MESSAGES:
		return false
	case CP_GPRS_MOBILITY_MANAGEMENT_MESSAGES:
		return false
	case CP_GPRS_SESSION_MANAGEMENT_MESSAGES:
		return false
	case CP_NON_CALL_RELATED_SS_MESSAGES:
		return false
	case CP_LOCATION_SERVICE_SPECIFITED:
		return false
	case CP_EXTENSION_OF_THE_PD_TO_ONE_OCTET_LENGTH:
		return false
	case CP_USED_BY_TESTS_PROCEDURES:
		return false
	case CP_SMS_MESSAGES:
		return true
	}

	return false
}

func (msg *Cpmessage) loadRpDuHead() {
	var tmp byte
	var size int

	tmp = msg.DataSource[msg.Lp]
	msg.LengthInd = tmp
	msg.Lp++

	size = msg.Lp + int(msg.LengthInd)
	msg.CpUserData = msg.DataSource[msg.Lp:size]

}
