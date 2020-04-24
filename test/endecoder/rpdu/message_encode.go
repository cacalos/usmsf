package rpdu

import (
	"encoding/binary"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/usmsf.git/endecoder/rpdu/encoders"
)

/********************************************************
define seo test code
*********************************************************/

func (msg *Rpmessage) RpEncode() (ret []byte) {
	var RpHead []byte
	var i int

	if msg.RpMessageType == RP_DATA_N_MS {
		//RP encode
		RpHead = msg.MakeRpDu()
		if msg.RpError != RP_SUCC {
			ulog.Error("Fail : MakeRpAddr() -Addr Result --------")
			return
		}
		for i = range RpHead {
			ret = append(ret, RpHead[i])
		}

		ulog.Info("Only RpEncoding Value :", RpHead)

		ret = append(ret, msg.RpUserDataLen)

		for i = range msg.RpUserData {
			ret = append(ret, msg.RpUserData[i])
		}

	} else {
		ulog.Error("Invalid MessageType")
	}

	return
}

func (msg *Rpmessage) RpEncodeAck() (ret []byte) {
	var RpCause byte
	var RpHead []byte
	var i int

	RpHead = msg.MakeRpDuAck()

	if msg.RpMessageType == RP_ACK_N_MS && msg.RpError == RP_SUCC {

		for i = range RpHead {
			ret = append(ret, RpHead[i])
		}

		if msg.RpLengthInd != 0 {
			ret = append(ret, msg.RpLengthInd)

			for i = range msg.Tpdu {
				ret = append(ret, msg.Tpdu[i])

			}
		}

	} else if msg.RpMessageType == RP_ERROR_N_MS && msg.RpError != RP_SUCC {

		RpCause = msg.RpError

		for i = range RpHead {
			ret = append(ret, RpHead[i])
		}

		RpErrLen := byte(binary.Size(RpCause))
		ret = append(ret, RpErrLen)
		ret = append(ret, RpCause)
		ret = append(ret, msg.RpDiagnostic)

		if msg.RpLengthInd != 0 {
			ret = append(ret, msg.RpLengthInd)

			for i = range msg.Tpdu {
				ret = append(ret, msg.Tpdu[i])

			}
		}

	} else {
		ulog.Error("Invalid MessageType")
		// err
		return
	}
	ulog.Info("Encoding RPACK(ERROR) : %02x", ret)
	return
}

func (msg *Rpmessage) MakeRpAddrDa() (ret []byte) {
	// RP-DA

	var RpAddrLen byte
	var RpTonNpi byte
	var RpAddr []byte
	var RpTon byte
	var i int

	RpAddrLen = byte(msg.RpDestAddrLen)

	if RpAddrLen%2 == 0 {
		RpAddrLen = (RpAddrLen / 2) + 1 //ton/npi
		ret = append(ret, RpAddrLen)
	} else {
		RpAddrLen = (RpAddrLen / 2) + 2 //ton/npi + last num
		ret = append(ret, RpAddrLen)
	}

	RpTon = 1
	RpTon = RpTon << 7
	RpTonNpi = RpTon | (msg.RpDestTon << 4)
	RpTonNpi = RpTonNpi | (msg.RpDestNpi & 0x0F)
	ret = append(ret, RpTonNpi)

	RpAddr = msg.makeRpDa()
	if msg.RpError != RP_SUCC {
		ulog.Error("Fail : the reason is MakeRpda")
		return
	}

	for i = range RpAddr {
		ret = append(ret, RpAddr[i])
	}

	return

}

func (msg *Rpmessage) MakeRpAddrOa() (ret []byte) {
	// RP-DA

	var RpAddrLen byte
	var RpTonNpi byte
	var RpAddr []byte
	var RpTon byte
	var i int

	//	ret = bytes.NewBufferString(``)

	//	if msg.Dir == DIRECTION_N_MS {
	RpAddrLen = byte(msg.RpOrigAddrLen)

	if RpAddrLen%2 == 0 {
		RpAddrLen = (RpAddrLen / 2) + 1 //ton/npi
		//	ret.WriteString(hex.EncodeToString([]byte{RpAddrLen})) //Cp 1byte
		ret = append(ret, RpAddrLen)
	} else {
		RpAddrLen = (RpAddrLen / 2) + 2 //ton/npi + last num
		ret = append(ret, RpAddrLen)
	}

	RpTon = 1
	RpTon = RpTon << 7
	RpTonNpi = RpTon | (msg.RpOrigTon << 4)
	RpTonNpi = RpTonNpi | (msg.RpOrigNpi & 0x0F)
	ret = append(ret, RpTonNpi)
	//	ret.WriteString(hex.EncodeToString([]byte{RpTonNpi})) //Cp 1byte

	RpAddr = msg.makeRpOa()
	if msg.RpError != RP_SUCC {
		ulog.Error("Fail : the reason is MakeRpda")
		return
	}

	for i = range RpAddr {
		ret = append(ret, RpAddr[i])
	}

	return
}

// Header information, part one
func (msg *Rpmessage) MakeRpDuAck() (ret []byte) {
	ret = append(ret, msg.RpMessageType)
	ret = append(ret, msg.RpMessageReference)

	return
}

// Header information, part one
func (msg *Rpmessage) MakeRpDu() (ret []byte) {
	var i int

	ret = append(ret, msg.RpMessageType)
	ret = append(ret, msg.RpMessageReference)

	//Rp OA
	if msg.Dir != DIRECTION_MS_N {
		var RpOa []byte

		RpOa = msg.MakeRpAddrOa()
		if msg.RpError != RP_SUCC {
			ulog.Error("Fail : MakeRpAddr() - OA")
			return
		}

		for i = range RpOa {
			ret = append(ret, RpOa[i])
		}

	} else {
		var RpOa byte
		ret = append(ret, RpOa)
	}

	//Rp DA
	if msg.Dir != DIRECTION_N_MS {
		var RpDa []byte

		RpDa = msg.MakeRpAddrDa()
		if msg.RpError != RP_SUCC {
			ulog.Error("Fail : MakeRpAddr() - DA")
			return
		}

		for i = range RpDa {
			ret = append(ret, RpDa[i])
		}
	} else {
		var RpDa byte
		ret = append(ret, RpDa)
	}

	return
}

// Originating Address
func (msg *Rpmessage) makeRpDa() (ret []byte) {
	var num uint64
	var buf []byte
	var i int

	num, msg.RpError = UParseUint(msg.RpDestAddr, 0, 64) //0109595 예시에서, 앞자리가 0일때, 8진수로 인식, 에러남.

	if msg.RpError != RP_SUCC {
		ulog.Error("Fail : err strconv.UParseUint ")
		return
	}

	buf = encoders.NewSemiOctet().Encode(msg.RpDestAddr, num)
	if len(buf) < int(msg.RpDestAddrLen) {
		var nb []byte
		// first zero
		//		for i = 0; i < int(msg.RpDestAddrLen)/2-len(buf)+(int(msg.RpDestAddrLen)/2)%2; i++ {
		//			nb = append(nb, 0x0)
		//		}
		nb = append(nb, buf...)
		buf = nb

	}
	i = 0
	for i = range buf {
		ret = append(ret, buf[i])
	}

	return
}

// Originating Address
func (msg *Rpmessage) makeRpOa() (ret []byte) {
	var num uint64
	var buf []byte
	var i int

	num, msg.RpError = UParseUint(msg.RpOrigAddr, 0, 64) //0109595 예시에서, 앞자리가 0일때, 8진수로 인식, 에러남.

	if msg.RpError != RP_SUCC {

		ulog.Error("Fail : err strconv.UParseUint ")
		return
	}
	buf = encoders.NewSemiOctet().Encode(msg.RpOrigAddr, num)
	if len(buf) < int(msg.RpOrigAddrLen) {
		var nb []byte
		// first zero
		//		for i = 0; i < int(msg.RpOrigAddrLen)/2-len(buf)+(int(msg.RpOrigAddrLen)/2)%2; i++ {
		//			nb = append(nb, 0x0)
		//		}
		nb = append(nb, buf...)
		buf = nb

	}
	i = 0
	for i = range buf {
		ret = append(ret, buf[i])
	}

	return
}

// ParseUint is like ParseInt but for unsigned numbers.
func UParseUint(s string, base int, bitSize int) (ret uint64, err byte) {
	const intSize = 32 << (^uint(0) >> 63)

	// IntSize is the size in bits of an int or uint value.
	const IntSize = intSize
	const maxUint64 = (1<<64 - 1)

	if len(s) == 0 {
		err = RP_INVALID_MANDATORY_INFORMATION
		ret = 0
		return
	}

	//	s0 := s
	switch {

	case base == 0:
		// Look for octal, hex prefix.
		switch {
		/*
			case s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X'):
				if len(s) < 3 {
					return 0, syntaxError(fnParseUint, s0)
				}
				base = 16
				s = s[2:]
			case s[0] == '0':
				base = 8
				s = s[1:]
		*/
		case s[0] == '0':
			base = 10
		default:
			base = 10
		}

	default:
		ret = 0
		err = RP_INVALID_MANDATORY_INFORMATION
		return
	}

	if bitSize == 0 {
		bitSize = int(IntSize)
	} else if bitSize < 0 || bitSize > 64 {
		err = RP_INVALID_MANDATORY_INFORMATION
		ret = 0
		return
	}

	// Cutoff is the smallest number such that cutoff*base > maxUint64.
	// Use compile-time constants for common cases.
	var cutoff uint64
	switch base {
	case 10:
		cutoff = maxUint64/10 + 1
	default:
		cutoff = maxUint64/uint64(base) + 1
	}

	maxVal := uint64(1)<<uint(bitSize) - 1

	var n uint64
	for _, c := range []byte(s) {
		var d byte
		switch {
		case '0' <= c && c <= '9':
			d = c - '0'
		case 'a' <= c && c <= 'z':
			d = c - 'a' + 10
		case 'A' <= c && c <= 'Z':
			d = c - 'A' + 10
		default:
			err = RP_INVALID_MANDATORY_INFORMATION
			ret = 0
			return
		}
		if d >= byte(base) {
			err = RP_INVALID_MANDATORY_INFORMATION
			ret = 0
			return
		}

		if n >= cutoff {
			// n*base overflows
			err = RP_INVALID_MANDATORY_INFORMATION
			ret = maxVal
			return
		}
		n *= uint64(base)

		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			// n+v overflows
			err = RP_INVALID_MANDATORY_INFORMATION
			ret = maxVal

			return
		}
		n = n1
	}

	ret = n
	err = 0

	return
}
