package rpdu

import (
	//"bytes"
	"container/list"
	"io"
	//	"regexp"
	"sync"
)

const _MaxBytes = int(140)
const start = string("Function Start")

/****************************************************
Insert seo test code
*****************************************************/
const (
	//Direction
	DIRECTION_N_MS = 1
	DIRECTION_MS_N = 2

	//ton
	TON_UNKNOWN       = 0
	TON_INTERNATIONAL = 1
	TON_NATIONAL      = 2

	//npi
	NPI_UNKNOWN = 0
	NPI_E164    = 1

	//RP_MTI
	RP_DATA_MS_N = 0x00 /* ms -> n :  RP DATA	*/
	//	RP_RESERVED_N_MS = 0x00 /* n -> ms :  Reserved	*/
	//	RP_RESERVED_MS_N = 0x01 /* ms -> n : Reserved	*/
	RP_DATA_N_MS = 0x01 /* n -> ms :  RP DATA	*/
	RP_ACK_MS_N  = 0x02 /* ms -> n :  RP ACK	*/
	//	RP_RESERVED_N_MS = 0x02 /* n -> ms :  Reserved	*/
	//	RP_RESERVED_MS_N = 0x03 /* ms -> n : Reserved	*/
	RP_ACK_N_MS   = 0x03 /* n -> ms : RP ACK	*/
	RP_ERROR_MS_N = 0x04 /* ms -> n : RP ERROR	*/
	//	RP_RESERVED_N_MS = 0x04 /* n -> ms :  Reserved	*/
	//	RP_RESERVED_MS_N = 0x05 /* ms-> n  : Reserved	*/
	RP_ERROR_N_MS = 0x05 /* n -> ms :  RP ERROR	*/
	RP_SMMA_MS_N  = 0x06 /* ms-> n  :  RP SMMA	*/
	//	RP_RESERVED_MS_N = 0x06 /* n -> ms :  Reserved	*/
	//	RP_RESERVED_MS_N = 0x07 /* ms -> n : Reserved	*/
	//	RP_RESERVED_MS_N = 0x07 /* n -> ms : Reserved	*/
	RP_INVALID_TYPE = 0xFF

	// RP_ERROR_CODE {
	RP_SUCC = 0x00

	// MO
	RP_UNASSIGNED_NUMBER                  = 1
	RP_OPERATOR_DETERMINED_BARRING        = 8
	RP_CALL_BARRED                        = 10
	RP_RESERVED                           = 11
	RP_SHORT_MESSAGE_TRANSFER_REJECTED    = 21
	RP_DESTINATION_OUT_OF_ORDER           = 27
	RP_UNIDENTIFIED_SUBSCRIBER            = 28
	RP_FACILITY_REJECTED                  = 29
	RP_UNKNOWN_SUBSCRIBER                 = 30
	RP_NETWORK_OUT_OF_ORDER               = 38
	RP_TEMPORARY_FAILURE                  = 41
	RP_CONGESTION                         = 42
	RP_RESOURCES_UNAVAILABLE_UNSPECIFIED  = 47
	RP_REQUESTED_FACILITY_NOT_SUBSCRIBED  = 50
	RP_REQUESTED_FACILITY_NOT_IMPLEMENTED = 69
	RP_INTERWORKING_UNSPECIFIED           = 127

	//}

	// MT {
	RP_MEMORY_CAPACITY_EXCEEDED = 22
	//}

	// MO/MT COMMON {
	RP_INVALID_SHORT_MESSAGE_TRANSFER_REFERENCE_VALUE           = 81
	RP_SEMANTICALLY_INCORRECT_MESSAGE                           = 95
	RP_INVALID_MANDATORY_INFORMATION                            = 96
	RP_MESSAGE_TYPE_NON_EXISTENT_OR_NOT_IMPLEMENTED             = 97
	RP_MESSAGE_NOT_COMPATIBLE_WITH_SHORT_MESSAGE_PROTOCOL_STATE = 98
	RP_INFORMATION_ELEMENT_NON_EXISTENT_OR_NOT_IMPLEMENTED      = 99
	RP_PROTOCOL_ERROR_UNSPECIFIED                               = 111

	//}
)

type RPDU struct {
	Direction        byte
	MessageType      byte
	MessageReference byte

	RData  RpData
	RAck   RpAck
	RError RpError

	RpError byte
}

type RAddress struct {
	//	min [25]byte
	Min string
	Ton byte
	Npi byte
	Len int
}

type RpCause struct {
	//	ElementId  byte
	Len        byte
	Cause      byte
	Diagnostic byte
}

type RpData struct {
	OrigAddr     RAddress
	DestAddr     RAddress
	RpDataLength byte
	RpUserData   [232]byte
}

type RpAck struct {
	//	ElementId    byte
	RpDataLength byte
	RpUserData   [232]byte
}

type RpError struct {
	CauseCode RpCause
	//	ElementId    byte
	RpDataLength byte
	RpUserData   [232]byte
}

// Decoded sms message
type Rpmessage struct {
	//// 자료형은 그냥 임시로 작성 중... 필요한대로 수정 필요
	Dir                byte
	RpMessageType      byte
	RpMessageReference byte

	RpOrigAddr       string
	RpOrigTon        byte
	RpOrigNpi        byte
	RpOrigTypeSource byte
	RpOrigAddrLen    uint8

	RpDestAddr       string
	RpDestTon        byte
	RpDestNpi        byte
	RpDestTypeSource byte
	RpDestAddrLen    uint8

	RpError byte

	RpDiagnostic byte

	Lp uint8

	RpUserDataLen byte

	//	RpUserData string
	RpUserData []byte

	DataSource []byte // Source pdu data
	//	Err        error  // Last error
	End bool // Decoding of message completed

	RpLengthInd byte
	Tpdu        []byte
}

// Message SMS message
type RpMessage interface {
	//	Error() error
	Complete() bool
	Direction() byte

	RpMsgType() byte
	RpMsgRef() byte

	RpOrigAddr_f() string
	RpOrigTon_f() byte
	RpOrigNpi_f() byte
	RpOrigAddrLen_f() uint8

	RpDestAddr_f() string
	RpDestTon_f() byte
	RpDestNpi_f() byte
	RpDestAddrLen_f() uint8
}

type RpFnDecoder func(RpMessage)

type rpimpl struct {
	RpdoCloseUp         chan bool      // Begin shutdown decoder goroutine
	RpdoCloseDone       sync.WaitGroup // Sync/wait when goroutine is running
	RpdoCount           sync.WaitGroup // Consideration received and processed messages
	RpDec               chan []byte    // Channel for decoder
	RpDecFn             RpFnDecoder    // Function call after new message decoded
	RpIncomleteMessages *list.List     // Temporary storage of partially received SMS messages

	//seo test
	RpDecVal chan *Rpmessage // Channel for decoder
	RpData   *Rpmessage
}

// Interface is an interface
type RpInterface interface {
	// Done Waiting for processing all incoming messages
	RpDone()
	// Writer Return writer
	RpWriter() io.Writer
	// Encoder SMS encoder

	/************************************
	  define seo test code
	  *************************************/
	RpEncoder(RPDU) ([]byte, byte)
	RpEncoderAck(RPDU) ([]byte, byte)
	RpDecoder(fn RpFnDecoder) RpInterface
	Rpdecode() *Rpmessage
}
