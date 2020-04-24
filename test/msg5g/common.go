package msg5g

import (
	"math/rand"

	"camel.uangel.com/ua5g/ulib.git/problem"
)

const (
	SMS_DELIVERY_PENDING  = "SMS_DELIVERY_PENDING"
	SMS_DELIVERY_COMPLETE = "SMS_DELIVERY_COMPLETED"
	SMS_DELIVERY_FAILED   = "SMS_DELIVERY_FAILED"

	SMSF_BOUNDARY = "SMSF_Boundary"

	SMSF_ERR   = -1
	SMSF_NOTOK = 0
	SMSF_OK    = 1

	ADD     = "ADD"
	MOVE    = "MOVE"
	REMOVE  = "REMOVE"
	REPLACE = "REPLACE"
)

// URI URI
type URI string

// IPv4Addr IPv4Addr
type IPv4Addr string

// IPv6Addr IPv6Addr
type IPv6Addr string

// DateTime DateTime
type DateTime string

// Fqdn Fully Qualified Domain Name
type Fqdn string

// Dnn DNN ????
type Dnn string

// UInteger Unsigned Integer ????
type UInteger uint32

// SupportedFeatures SupportedFeatures
type SupportedFeatures string

type ChangeType string

// PlmnID PlmnID
type PlmnID struct {
	Mcc string `json:"mcc"`
	Mnc string `json:"mnc"`
}

// Snssai Snssai
type Snssai struct {
	Sst UInteger `json:"sst" valid:"required"`
	Sd  *string  `json:"sd,omitempty"`
}

// Guami Guami
type Guami struct {
	PlmnID PlmnID `json:"plmnId" valid:"required"`
	AmfID  AmfID  `json:"amfId" valid:"required"`
}

// AmfID AMF ID
type AmfID string //'^[A-Fa-f0-9]{6}$'

// Tai Tai
type Tai struct {
	PlmnID PlmnID `json:"plmnId" valid:"required"`
	Tac    Tac    `json:"tac" valid:"required"`
}

type SmsfDiameterAddress struct {
	Name  string `json:"name"`
	Realm string `json:"realm"`
}

type InvalidParams struct {
	Param  string `json:"param"`
	Reason string `json:"reason"`
}

type SingleNssai struct {
	Sst uint   `json:"sst"`
	Sd  string `json:"sd"`
}

type Tac string // '(^[A-Fa-f0-9]{4}$)|(^[A-Fa-f0-9]{6}$)'
// EutraCellID EutraCellID

type GNbId struct {
	BitLength int    `json:"bitLength"`
	GNbValue  string `json:"gNbValue"`
}

type GlobalRanNodeId struct {
	PlmnID  PlmnID `json:"plmnId`
	nwlwfld string `json:"n3IwfId,omitempty"`
	GNbId   GNbId  `json"gNbId"`
	NgeNbId string `json:"ngeNbId"`
}

type RefToBinaryData struct {
	ContentID string `json:"contentId"`
}

type N1MessageContainer struct {
	N1MessageClass   string          `json:"n1MessageClass,omitempty"`
	N1MessageContent RefToBinaryData `json:"n1MessageContent,omitempty"`
	NfId             string          `json:"nfId,omitempty"`
}

type EutraLocation struct {
	Tai                      Tai             `json:"tai"`
	Ecgi                     Ecgi            `json:"ecgi"`
	AgeOfLocationInformation int             `json:"ageOfLocationInformation,omitempty"`
	UeLocationTimestamp      string          `json:"ueLocationTimesamp,omitempty"`
	GeographicalInformation  string          `json:"geographicalInformation,omitempty"`
	GeodeticInformation      string          `json:"geodeticInformation,omitempty"`
	GlobalNgenbId            GlobalRanNodeId `json:"globalNgenbId,omitempty"`
}

type NrLocation struct {
	Tai                      Tai             `json:"tai"`
	Ncgi                     Ncgi            `json:"ncgi"`
	AgeOfLocationInformation int             `json:"ageOfLocationInformation,omitempty"`
	UeLocationTimestamp      string          `json:"ueLocationTimesamp,omitempty"`
	GeographicalInformation  string          `json:"geographicalInformation,omitempty"`
	GeodeticInformation      string          `json:"geodeticInformation,omitempty"`
	GlobalNgenbId            GlobalRanNodeId `json:"globalNgenbId,omitempty"`
}

type N3gaLocation struct {
	N3gppTai   Tai    `json:"tai"`
	N3lwfId    string `json:"n3IwfId"`
	UeIpv4Addr string `json:"ueIpv4Addr"`
	UeIpv6Addr string `json:"ueIpv6Addr"`
	PortNumber uint   `json:"portNumber"`
}

type UserLocation struct {
	EutraLocation EutraLocation `json:"eutraLocation"`
	NrLocation    NrLocation    `json:"nrLocation:"`
	N3gaLocation  N3gaLocation  `josn:"n3gaLocation"`
}

// TaiRange TaiRange - TS 29.510
type TaiRange struct {
	PlmnID       PlmnID      `json:"plmnId,omitempty"`
	TacRangeList []*TacRange `json:"tacRangeList,omitempty"`
}

// Ecgi Ecgi
type Ecgi struct {
	PlmnID      PlmnID      `json:"plmnId" valid:"required"`
	EutraCellID EutraCellID `json:"eutraCellId" valid:"required"`
}

// EutraCellID EutraCellID
type EutraCellID string // '^[A-Fa-f0-9]{7}$'

// Ncgi Ncgi
type Ncgi struct {
	PlmnID   PlmnID   `json:"plmnId" valid:"required"`
	NrCellID NrCellID `json:"nrCellId" valid:"required"`
}

// NrCellID NrCellID
type NrCellID string //'^[A-Fa-f0-9]{9}$

// N2InterfaceAmfInfo N2InterfaceAmfInfo - TS 29.510
type N2InterfaceAmfInfo struct {
	IPv4EndpointAddress []*IPv4Addr `json:"ipv4EndpointAddress,omitempty"`
	IPv6EndpointAddress []*IPv6Addr `json:"ipv6EndpointAddress,omitempty"`
	AmfName             *string     `json:"amfName,omitempty"`
}

// PcfInfo PcfInfo
type PcfInfo struct {
	DnnList       []Dnn        `json:"amfSetId" valid:"omitempty"`
	SupiRangeList []*SupiRange `json:"supiRangeList,omitempty"`
}

// BsfInfo BsfInfo
type BsfInfo struct {
	IPv4AddressRanges []*IPv4AddressRange `json:"ipv4AddressRanges,omitempty"`
	IPv6PrefixRanges  []*IPv6PrefixRange  `json:"ipv6PrefixRanges,omitempty"`
}

// IPv4AddressRange IPv4AddressRange
type IPv4AddressRange struct {
	Start *IPv4Addr `json:"start,omitempty"`
	End   *IPv4Addr `json:"end,omitempty"`
}

// IPv6PrefixRange IPv6PrefixRange
type IPv6PrefixRange struct {
	Start *IPv6Prefix `json:"start,omitempty"`
	End   *IPv6Prefix `json:"end,omitempty"`
}

// IPv6Prefix IPv6Prefix
type IPv6Prefix string

// UpfInfo UpfInfo
type UpfInfo struct {
	SnssaiUpfInfoList    []*SnssaiUpfInfoItem    `json:"sNssaiUpfInfoList" valid:"required"`
	SmfServingArea       []string                `json:"smfServingArea,omitempty"`
	InterfaceUpfInfoList []*InterfaceUpfInfoItem `json:"interfaceUpfInfoList,omitempty"`
}

// SnssaiUpfInfoItem SnssaiUpfInfoItem
type SnssaiUpfInfoItem struct {
	Snssai         Snssai            `json:"sNssai" valid:"required"`
	DnnUpfInfoList []*DnnUpfInfoItem `json:"dnnUpfInfoList" valid:"required"`
}

// DnnUpfInfoItem DnnUpfInfoItem
type DnnUpfInfoItem struct {
	Dnn Dnn `json:"dnn" valid:"required"`
}

// InterfaceUpfInfoItem InterfaceUpfInfoItem - TS 29.510
type InterfaceUpfInfoItem struct {
	InterfaceType       string     `json:"interfaceType" valid:"required"`
	IPv4EndpointAddress []IPv4Addr `json:"ipv4EndpointAddress,omitempty"`
	IPv6EndpointAddress []IPv6Addr `json:"ipv6EndpointAddress,omitempty"`
	EndpointFqdn        *Fqdn      `json:"endpointFqdn,omitempty"`
	NetworkInstance     *string    `json:"networkInstance,omitempty"`
}

// TacRange TacRange
type TacRange struct {
	Start   *string `json:"start,omitempty"`
	End     *string `json:"end,omitempty"`
	Pattern *string `json:"pattern,omitempty"`
}

// SmfInfo SmfInfo
type SmfInfo struct {
	DnnList      []Dnn       `json:"amfSetId" valid:"required"`
	TaiList      []*Tai      `json:"taiList,omitempty"`
	TaiRagneList []*TaiRange `json:"taiRangeList,omitempty"`
	PgwFqdn      *Fqdn       `json:"pgwFqdn,omitempty"`
}

// UdmInfo UdmInfo - TS 29.510
type UdmInfo struct {
	GroupID               *string          `json:"groupId,omitempty"`
	SupiRanges            []*SupiRange     `json:"supiRanges,omitempty"`
	GpsiRanges            []*IdentityRange `json:"gpsiRanges,omitempty"`
	ExternalGroupIDRanges []*IdentityRange `json:"externalGroupIdentifiersRanges,omitempty"`
	RoutingIndicators     []string         `json:"routingIndicators,omitempty"`
}

// AusfInfo AusfInfo - TS 29.510
type AusfInfo struct {
	GroupID           *string      `json:"groupId,omitempty"`
	SupiRanges        []*SupiRange `json:"supiRanges,omitempty"`
	RoutingIndicators []string     `json:"routingIndicators,omitempty"`
}

// AmfInfo AmfInfo - TS 29.510
type AmfInfo struct {
	AmfSetID             string              `json:"amfSetId" valid:"required"`
	AmfRegionID          string              `json:"amfRegionId" valid:"required"`
	GuamiList            []*Guami            `json:"guamiList" valid:"required"`
	TaiList              []*Tai              `json:"taiList,omitempty"`
	TaiRagneList         []*TaiRange         `json:"taiRangeList,omitempty"`
	BackupInfoAmfFailure []*Guami            `json:"backupInfoAmfFailure,omitempty"`
	BackupInfoAmfRemoval []*Guami            `json:"backupInfoAmfRemoval,omitempty"`
	N2InterfaceAmfInfo   *N2InterfaceAmfInfo `json:"n2InterfaceAmfInfo,omitempty"`
}

// UdrInfo UdrInfo - TS 29.510
type UdrInfo struct {
	GroupID               *string          `json:"groupId,omitempty"`
	SupiRanges            []*SupiRange     `json:"supiRanges,omitempty"`
	GpsiRanges            []*IdentityRange `json:"gpsiRanges,omitempty"`
	ExternalGroupIDRanges []*IdentityRange `json:"externalGroupIdentifiersRanges,omitempty"`
	SupportedDataSets     []string         `json:"supportedDataSets,omitempty"`
}

// SupiRange SupiRange - TS 29.510
type SupiRange struct {
	Start   *string `json:"start,omitempty"`
	End     *string `json:"end,omitempty"`
	Pattern *string `json:"pattern,omitempty"`
}

// IdentityRange IdentityRange - TS 29.510
type IdentityRange struct {
	Start   *string `json:"start,omitempty"`
	End     *string `json:"end,omitempty"`
	Pattern *string `json:"pattern,omitempty"`
}

// BackupAmfInfo - TS 29.571
type BackupAmfInfo struct {
	BackupAmf string   `json:"backupAmf"`
	GuamiList []*Guami `json:"guamiList" valid:"required"`
}

// TraceData - TS 29.571
type TraceData struct {
	TraceBuf                 string `json:"traceBuf"`
	TraceDepth               int    `json:"traceDepth"`
	NeTypeList               string `json:"neTypeList"`
	EventList                string `json:"eventList"`
	CollectionEntityIpv4Addr string `json:"collectionEntityIpv4Addr,omitempty"`
	CollectionEntityIpv6Addr string `json:"collectionEntityIpv6Addr,omitempty"`
	InterfaceList            string `json:"interfaceList,omitempty"`
}

type NotifyItem struct {
	ResourceId URI           `json:"resourceId"`
	Changes    []*ChangeItem `json:"changes"`
}

type ChangeItem struct {
	Op        ChangeType  `json:"op" valid:"required"`
	Path      string      `json:"path" valid:"required"`
	From      *string     `json:"from,omitempty"`
	OrigValue interface{} `json:"origValue,omitempty"`
	NewValue  interface{} `json:"newValue,omitempty"`
}

type Nfinterface interface {
	Make() ([]byte, error)
}

// InvalidParam is problem.InvalidParam
type InvalidParam = problem.InvalidParam

// ProblemDetails is problem.Details
type ProblemDetails = problem.Details

// JSONParseError is problem.JSONParseError
var JSONParseError = problem.JSONParseError

// NotSupported is problem.NotSupported
var NotSupported = problem.NotSupported

// SystemError is problem.SystemError
var SystemError = problem.SystemError

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

//RandASCIIBytes - A helper function create and fill a slice of length n with characters from a-zA-Z0-9_-. It panics if there are any problems getting random bytes.
func RandASCIIBytes(n int) []byte {
	output := make([]byte, n)

	// We will take n bytes, one byte for each character of output.
	randomness := make([]byte, n)

	// read all random
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}

	l := len(letterBytes)
	// fill output
	for pos := range output {
		// get random item
		random := uint8(randomness[pos])

		// random % 64
		randomPos := random % uint8(l)

		// put into output
		output[pos] = letterBytes[randomPos]
	}

	return output
}
