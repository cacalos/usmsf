package msg5g

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/go-openapi/strfmt"
)

// NFStatus values
const (
	NFStatusRegistered     = "REGISTERED"
	NFStatusSuspended      = "SUSPENDED"
	NFStatusUndisdoverable = "UNDISCOVERABLE"
)

// NFMgmtProfile NF 관리 API에서 사용되는 NF 프로파일 타입
type NFMgmtProfile struct {
	NfInstanceID        string           `json:"nfInstanceId" valid:"required"`
	NfType              string           `json:"nfType" valid:"required"`
	NfStatus            string           `json:"nfStatus" valid:"required"`
	HbTimer             *int             `json:"heartBeatTimer,omitempty"`
	Plmn                *PlmnID          `json:"plmn,omitempty"`
	Snssais             []*Snssai        `json:"sNssais,omitempty"`
	NsiList             []string         `json:"nsiList,omitempty"`
	Fqdn                *Fqdn            `json:"fqdn,omitempty"`
	InterPlmnFqdn       *Fqdn            `json:"interPlmnFqdn,omitempty"`
	IPv4Addresses       []IPv4Addr       `json:"ipv4Addresses,omitempty"`
	IPv6Addresses       []IPv6Addr       `json:"ipv6Addresses,omitempty"`
	Priority            *int             `json:"priority,omitempty"`
	Capacity            *int             `json:"capacity,omitempty"`
	Load                *int             `json:"load,omitempty"`
	Locality            *string          `json:"locality,omitempty"`
	UdrInfo             *UdrInfo         `json:"udrInfo,omitempty"`
	UdmInfo             *UdmInfo         `json:"udmInfo,omitempty"`
	AusfInfo            *AusfInfo        `json:"ausfInfo,omitempty"`
	AmfInfo             *AmfInfo         `json:"amfInfo,omitempty"`
	SmfInfo             *SmfInfo         `json:"smfInfo,omitempty"`
	UpfInfo             *UpfInfo         `json:"upfInfo,omitempty"`
	PcfInfo             *PcfInfo         `json:"pcfInfo,omitempty"`
	BsfInfo             *BsfInfo         `json:"bsfInfo,omitempty"`
	CustomInfo          interface{}      `json:"customInfo,omitempty"`
	RecoveryTime        *strfmt.DateTime `json:"recoveryTime,omitempty"`
	NfServices          interface{}      `json:"nfServices,omitempty"`
	NfProfileChangesInd *bool            `json:"nfProfileChangesInd,omitempty"`
}

// NFPatchData NF 관리 중 갱신 API의 JSON Body 타입
type NFPatchData struct {
	From  string      `json:"from,omitempty"`
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

//NFDiscService NF Discovery 서비스에서 사용되는 Service 타입
type NFDiscService struct {
	ServiceInstanceID string              `json:"serviceInstanceId" valid:"required"`
	ServiceName       string              `json:"serviceName" valid:"required"`
	Versions          []*NFServiceVersion `json:"versions" valid:"required"`
	Scheme            string              `json:"scheme" valid:"required"`
	NfServiceStatus   string              `json:"nfServiceStatus" valid:"required"`
	Fqdn              *Fqdn               `json:"fqdn,omitempty"`
	IPEndPoints       []*IPEndPoint       `json:"ipEndPoints,omitempty"`
	APIPrefix         *string             `json:"apiPrefix,omitempty"`
	DfltNotiSubss     []*DfltNotiSubs     `json:"defaultNotificationSubscriptions,omitempty"`
	Priority          *int                `json:"priority,omitempty"`
	Capacity          *int                `json:"capacity,omitempty"`
	Load              *int                `json:"load,omitempty"`
	RecoveryTime      *string             `json:"recoveryTime,omitempty"`
	SupportedFeatures *SupportedFeatures  `json:"supportedFeatures,omitempty"`
}

// NFServiceVersion NFServiceVersion
type NFServiceVersion struct {
	APIVersionInURI string    `json:"apiVersionInUri" valid:"required"`
	APIFullVersion  string    `json:"apiFullVersion" valid:"required"`
	Expiry          *DateTime `json:"expiry,omitempty"`
}

// SubscriptionData SubscriptionData - TS 29.510 V15.3.0
/* 참ê 'subscrCond' ? ?¸리뷰트의 값? ?¤음 중 ??가 ?  ? ??:
NfInstanceIdCond, NfTypeCond, AmfCond, GuamiListCond, NetworkSliceCond 및
NfGroupCond ??.
BSF ? PcfBinding ?±ë? 마?? PCF? instance ID 값? ?¬용하???? 감?ë   ?? 구독? NRF? ?청?? 구조?´므로, NfInstanceIdCond ??? ?¬ì
*/
type SubscriptionData struct {
	NfStatusNotificationURI URI                     `json:"nfStatusNotificationUri" valid:"required"`
	SubscrCond              *NfInstanceIDCond       `json:"subscrCond,omitempty"`
	SubscriptionID          *string                 `json:"subscriptionId,omitempty"`
	ValidityTime            *strfmt.DateTime        `json:"validityTime,omitempty"` //TODO: ?????
	ReqNotifEvents          []NotificationEventType `json:"reqNotifEvents,omitempty"`
	//ReqNfType               *NFType                 `json:"reqNfType,omitempty"`
	ReqNfFqdn *Fqdn   `json:"reqNfFqdn,omitempty"`
	PlmnID    *PlmnID `json:"plmnId,omitempty"`
	//NotifCondition          *NotifCondition         `json:"notifCondition,omitempty"`
}

// NfInstanceIDCond NfInstanceIDCond
type NfInstanceIDCond struct {
	NfInstanceID string `json:"nfInstanceId" valid:"required"`
}

// IPEndPoint IPEndPoint
type IPEndPoint struct {
	IPv4Address *IPv4Addr `json:"ipv4Address,omitempty"`
	IPv6Address *IPv6Addr `json:"ipv6Address,omitempty"`
	Transport   *string   `json:"transport,omitempty"`
	Port        *int      `json:"port,omitempty"`
}

// DfltNotiSubs DfltNotiSubs
type DfltNotiSubs struct {
	NotificationType   string  `json:"notificationType" valid:"required"`
	CallbackURI        URI     `json:"callbackUri" valid:"required"`
	N1MessageClass     *string `json:"n1MessageClass,omitempty"`
	N2InformationClass *string `json:"n2InformationClass,omitempty"`
}

// SearchResult NF Discovery 서비스 호출 후 검색된 NF 프로파일 정보를 담은 JSON Body 타입
type SearchResult struct {
	ValidityPeriod int              `json:"validityPeriod" valid:"required"`
	NfInstances    []*NFDiscProfile `json:"nfInstances,omitempty"`
}

// NFDiscProfile Discovery 서비스에서 사용되는 NF 프로파일 타입
type NFDiscProfile struct {
	NfInstanceID        string           `json:"nfInstanceId" valid:"required"`
	NfType              string           `json:"nfType" valid:"required"`
	NfStatus            string           `json:"nfStatus" valid:"required"`
	Plmn                *PlmnID          `json:"plmn,omitempty"`
	Snssais             []*Snssai        `json:"sNssais,omitempty"`
	NsiList             []string         `json:"nsiList,omitempty"`
	Fqdn                *Fqdn            `json:"fqdn,omitempty"`
	IPv4Addresses       []IPv4Addr       `json:"ipv4Addresses,omitempty"`
	IPv6Addresses       []IPv6Addr       `json:"ipv6Addresses,omitempty"`
	Priority            *int             `json:"priority,omitempty"`
	Capacity            *int             `json:"capacity,omitempty"`
	Load                *int             `json:"load,omitempty"`
	Locality            *string          `json:"locality,omitempty"`
	UdrInfo             *UdrInfo         `json:"udrInfo,omitempty"`
	UdmInfo             *UdmInfo         `json:"udmInfo,omitempty"`
	AusfInfo            *AusfInfo        `json:"ausfInfo,omitempty"`
	AmfInfo             *AmfInfo         `json:"amfInfo,omitempty"`
	SmfInfo             *SmfInfo         `json:"smfInfo,omitempty"`
	UpfInfo             *UpfInfo         `json:"upfInfo,omitempty"`
	PcfInfo             *PcfInfo         `json:"pcfInfo,omitempty"`
	BsfInfo             *BsfInfo         `json:"bsfInfo,omitempty"`
	CustomInfo          interface{}      `json:"customInfo,omitempty"`
	RecoveryTime        *strfmt.DateTime `json:"recoveryTime,omitempty"`
	NfServices          []*NFDiscService `json:"nfServices,omitempty"`
	NfProfileChangesInd *bool            `json:"nfProfileChangesInd,omitempty"`
}

// NotificationEventType NotificationEventType
// "NF_REGISTERED", "NF_DEREGISTERED", "NF_PROFILE_CHANGED"
type NotificationEventType string

// NotificationData NotificationData
type NotificationData struct {
	Event          NotificationEventType `json:"event" valid:"required"`
	NfInstanceURI  URI                   `json:"nfInstanceUri" valid:"required"`
	NfProfile      *NFMgmtProfile        `json:"nfProfile,omitempty"`
	ProfileChanges []*ChangeItem         `json:"profileChanges,omitempty"`
}

// LoadProfileFrom 지?? 경ë ?? BSF? Profile? 로?
func LoadProfileFrom(path string) (*NFMgmtProfile, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("The path is empty")
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	nfProfile := &NFMgmtProfile{}
	err = json.Unmarshal(bytes, nfProfile)
	if err != nil {
		return nil, err
	}

	return nfProfile, nil
}
