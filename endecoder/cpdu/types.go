package pdu

import (
	"container/list"
	"io"
	"regexp"
	"sync"
)

const _MaxBytes = int(140)

/****************************************************
Insert seo test code
*****************************************************/
const (
	//Direction
	DIRECTION_N_MS = 1
	DIRECTION_MS_N = 2

	// Incomming SMS: CP-DATA
	TypeCpData = 0x01
	// Incomming SMS: CP-ACK
	TypeCpAck = 0x04
	// Incomming SMS: CP-ERROR
	TypeCpError = 0x10

	//Protocol discriminator values
	CP_GORUP_CALL_CONTROL                      = 0x00 /* group call control */
	CP_BROADCAST_CALL_CONTROL                  = 0x01 /* broadcast call control */
	CP_EPS_SESSION_MANAGEMENT_MESSAGES         = 0x02 /* EPS session management messages */
	CP_CALL_CONTROL                            = 0x03 /* call control; call related SS messages */
	CP_GPRS_TRANSPARENT_TRANSPORT_PROTOCOL     = 0x04 /* GPRS Transparent Transport Protocol (GTTP) */
	CP_MOBILITY_MANAGEMENT_MESSAGE             = 0x05 /* mobility management messages */
	CP_RADIO_RESOURCES_MANAGEMENT_MESSAGES     = 0x06 /* radio resources management messages */
	CP_EPS_MOBILITY_MANAGEMENT_MESSAGES        = 0x07 /* EPS mobility management messages */
	CP_GPRS_MOBILITY_MANAGEMENT_MESSAGES       = 0x08 /* GPRS mobility management messages */
	CP_SMS_MESSAGES                            = 0x09 /* SMS messages */
	CP_GPRS_SESSION_MANAGEMENT_MESSAGES        = 0x0A /* GPRS session management messages */
	CP_NON_CALL_RELATED_SS_MESSAGES            = 0x0B /* non call related SS messages */
	CP_LOCATION_SERVICE_SPECIFITED             = 0x0C /* Location services specified in 3GPP TS 44.071 [8a] */
	CP_EXTENSION_OF_THE_PD_TO_ONE_OCTET_LENGTH = 0x0E /* extension of the PD to one octet length */
	CP_USED_BY_TESTS_PROCEDURES                = 0x0F /* used by tests procedures described in 3GPP TS 44.014 [5a], 3GPP TS 34.109 [17a], 3GPP TS 36.509 [26] and 3GPP TS 38.509 [29]. */

	//Transaction identifier Value
	CP_TRANSACTION_IDENTIFIER_FLAG0 = 0x00
	//	CP_TRANSACTION_IDENTIFIER_FLAG1 = 0x08

	//TI VALUE is 0
	CP_TI_VALUE_0_0   = 0x00
	CP_TI_VALUE_0_1   = 0x01
	CP_TI_VALUE_0_2   = 0x02
	CP_TI_VALUE_0_3   = 0x03
	CP_TI_VALUE_0_4   = 0x04
	CP_TI_VALUE_0_5   = 0x05
	CP_TI_VALUE_0_6   = 0x06
	CP_TI_TIE_0_VALUE = 0x07

	//TI VALUE is 8
	CP_TI_VALUE_8_0   = 0x08
	CP_TI_VALUE_8_1   = 0x09
	CP_TI_VALUE_8_2   = 0x0A
	CP_TI_VALUE_8_3   = 0x0B
	CP_TI_VALUE_8_4   = 0x0C
	CP_TI_VALUE_8_5   = 0x0D
	CP_TI_VALUE_8_6   = 0x0E
	CP_TI_TIE_8_VALUE = 0x0F

	/*
	 TI flag (octet 1)
	  Bit
	   8
	   0	The message is sent from the side that originates the TI
	   1	The message is sent to the side that originates the TI


	  TIO (octet 1)
	   Bits
	   7 6 5
	   0 0 0 	TI value 0
	   0 0 1        1
	   0 1 0        2
	   0 1 1        3
	   1 0 0        4
	   1 0 1        5
	   1 1 0        6
	   1 1 1 	The TI value is given by the TIE in octet 2
	*/

	//ton
	TON_UNKNOWN       = 0
	TON_INTERNATIONAL = 1
	TON_NATIONAL      = 2

	//npi
	NPI_UNKNOWN = 0
	NPI_E164    = 1

	//Cp Error Code
	CP_SUCC                                                         = 0x00 /*  00 successful */
	CP_NETWORK_FAILURE                                              = 0x11 /*  17   Network failure*/
	CP_CONGESTION                                                   = 0x16 /*  22   Congestion*/
	CP_INVALID_TRANSACTION_IDENTIFIER_VALUE                         = 0x51 /*  81   Invalid Transaction Identifier value */
	CP_SEMANTICALLY_INCORRECT_MESSAGE                               = 0x5F /*  95   Semantically incorrect message */
	CP_INVALID_MANDATORY_INFORMATION                                = 0x60 /*  96   Invalid mandatory information */
	CP_MESSAGE_TYPE_NON_EXISTENT_OR_NOT_IMPLEMENTED                 = 0x61 /*  97   Message type non existent or not implemented */
	CP_MESSAGE_NOT_COMPATIBLE_WITH_THE_SHORT_MESSAGE_PROTOCOL_STATE = 0x62 /*  98   Message not compatible with the short message protocol state */
	CP_INFORMATION_ELEMENT_NON_EXISTENT_OR_NOT_IMPLEMENTED          = 0x63 /*  99   Information element non existent or not implemented */
	CP_PROTOCOL_ERROR_UNSPECIFIED                                   = 0x6F /* 111   Protocol error, unspecified */

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

	// NumberTypeUnknown Unknown. The cellular network does not know what the format of a number
	NumberTypeUnknown = NumberType(`Unknown. The cellular network does not know what the format of a number`)
	// NumberTypeInternational International number format
	NumberTypeInternational = NumberType(`International number format`)
	// NumberTypeInternal Internal number of the country. The prefixes of the country have no numbers
	NumberTypeInternal = NumberType(`Internal number of the country`)
	// NumberTypeService The Service network number. Used by the operator.
	NumberTypeService = NumberType(`The Service network number`)
	// NumberTypeSubscriber The subscriber's number. Used when a certain idea of short number stored in one or more of the SC as part of a high-level application
	NumberTypeSubscriber = NumberType(`The subscriber's number`)
	// NumberTypeAlphanumeric Alphanumeric encoded in 7-bit encoding
	NumberTypeAlphanumeric = NumberType(`Alphanumeric encoded in 7-bit encoding`)
	// NumberTypeReduced Reduced number
	NumberTypeReduced = NumberType(`Reduced number`)
	// NumberTypeReserved Reserved
	NumberTypeReserved = NumberType(`Reserved`)

	// NumericPlanAlphanumeric Alphanumeric encoded
	NumericPlanAlphanumeric = NumberNumericPlan(`Alphanumeric encoded`)
	// NumericPlanInternational International
	NumericPlanInternational = NumberNumericPlan(`International`)
	// NumericPlanUnknown Unknown
	NumericPlanUnknown = NumberNumericPlan(`Unknown`)
)

type RPDU struct {
	Direction        byte
	MessageType      byte
	MessageReference byte

	RData  RpData
	RAck   RpAck
	RError RpError
}

type RAddress struct {
	//	min [25]byte
	Min string
	Ton byte
	Npi byte
	Len int
}

type RpCause struct {
	ElementId  byte
	Len        byte
	Cause      byte
	Diagnostic byte
}

type RpData struct {
	OrigAddr     RAddress
	DestAddr     RAddress
	RpDataLength byte
	RpUserData   [233]byte
}

type RpAck struct {
	ElementId    byte
	RpDataLength byte
	RpUserData   [233]byte
}

type RpError struct {
	CauseCode    RpCause
	ElementId    byte
	RpDataLength byte
	RpUserData   [233]byte
}

type CpEncode struct {
	Direction     byte
	ProtocolDiscr byte // 1/2
	TransactionId byte // 1/2
	MessageType   byte // 1
	LengthInd     byte //1
	CpData        [248]byte

	CpError byte

	CpUserData RPDU // 여기에 바로 접근해서 데이터 쓸수 있으면 좋은데.. 아직 어떻게 하는지 모르겠음..
	//	CpTestData    [249]byte  //test 용도... RPDU 만들어서 여기다 그냥 집어 넣을 생각으로 만들어 봄..
}

// Decoded sms message
type Cpmessage struct {
	//// 자료형은 그냥 임시로 작성 중... 필요한대로 수정 필요
	Dir           byte
	ProtocolDiscr byte // 1/2
	TransactionId byte // 1/2
	MessageType   byte // 1
	LengthInd     byte //1
	CpUserData    []byte

	CauseValue byte

	CpError byte

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

	Lp int

	RpDataLen byte

	RpUserData string

	DataSource []byte // Source pdu data
	//	Err        error  // Last error
	End bool // Decoding of message completed

	RpLengthInd byte
	Tpdu        []byte
}

// Message SMS message
type CpMessage interface {
	//	Error() error
	Complete() bool
	Direction() byte

	ProtocolDisc() byte
	TransactionID() byte
	MessageTypeInd() byte
	LengthIndicator() byte
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

	//	Make_and_send_Cp_resp() CpEncode
}

type CpFnDecoder func(CpMessage)

type cpimpl struct {
	CpdoCloseUp         chan bool      // Begin shutdown decoder goroutine
	CpdoCloseDone       sync.WaitGroup // Sync/wait when goroutine is running
	CpdoCount           sync.WaitGroup // Consideration received and processed messages
	CpDec               chan []byte    // Channel for decoder
	CpDecFn             CpFnDecoder    // Function call after new message decoded
	CpIncomleteMessages *list.List     // Temporary storage of partially received SMS messages

	//seo test
	CpDecVal chan *Cpmessage // Channel for decoder
	CpData   *Cpmessage
}

/*
type cpimpl struct {
	CpdoCloseUp         chan bool          // Begin shutdown decoder goroutine
	CpdoCloseDone       sync.WaitGroup     // Sync/wait when goroutine is running
	CpdoCount           sync.WaitGroup     // Consideration received and processed messages
	CpDec               chan *bytes.Buffer // Channel for decoder
	CpDecFn             CpFnDecoder        // Function call after new message decoded
	CpIncomleteMessages *list.List         // Temporary storage of partially received SMS messages

	//seo test
	CpDecVal chan *Cpmessage // Channel for decoder
	CpData   *Cpmessage
}
*/
// Interface is an interface
type CpInterface interface {
	// Done Waiting for processing all incoming messages
	CpDone()
	// Decoder Register function is invoked when decoding a new message
	//	Decoder(fn FnDecoder) Interface
	// Writer Return writer
	CpWriter() io.Writer
	// Encoder SMS encoder

	/************************************
	  define seo test code
	  *************************************/
	CpEncoder(CpEncode) ([]byte, byte)
	CpEncoderAck(CpEncode) ([]byte, byte)
	CpDecoder(fn CpFnDecoder) CpInterface
	Cpdecode() *Cpmessage
	ChanRet() chan *Cpmessage
	//CpEncoderTest(fn CpfnEncode) CpInterface
}

// NumberType Type of number
type NumberType string

// NumberNumericPlan Numbering plan identifier
type NumberNumericPlan string

var (
	rexDataWithCommand    = regexp.MustCompile(`^\+([0-9A-Za-z]+)\: (\d+),([^,]*),(\d+)[\t\n\f\r ]+`)
	rexDataWithoutCommand = regexp.MustCompile(`([0-9A-Fa-f]+)$`)
	rexNumeric            = regexp.MustCompile(`^([0-9]+)$`)
)
