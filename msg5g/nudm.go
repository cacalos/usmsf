package msg5g

import (
	//"fmt"
	//"sync"

	"fmt"

	jsoniter "github.com/json-iterator/go"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/utypes"
)

type UECMReq struct {
	SmsfInstanceID      string              `json:"smsfInstanceId"`
	SupportedFeatures   string              `json:"supported_features,omitempty"`
	PlmnID              PlmnID              `json:"plmnId"`
	SmsfMAPAddress      string              `json:"smsfMAPAddress,omitempty"`
	SmsfDiameterAddress SmsfDiameterAddress `json:"smsfDiameterAddress,omitempty"`
}

type UECMReqv1 struct {
	SmsfId            string `json:"smsfId"`
	SupportedFeatures string `json:"supported_features,omitempty"`
}

type NackResp struct {
	Type          string        `json:"type,omitempty"`
	Title         string        `json:"title,omitempty"`
	Status        int           `json:"status,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	Instance      string        `json:"instance,omitempty"`
	Cause         string        `json:"cause,omitempty"`
	InvalidParams InvalidParams `json:"invalidParams,omitempty"`
}

type SdmSubscribction struct {
	NfInstanceID          string      `json:"nfInstanceId"`
	ImplicitUnsubscribe   bool        `json:"implicitUnsubscribe,omitempty"`
	Expires               string      `json:"expires,omitempty"`
	CallbackReference     string      `json:"callbackReference"`
	MonitoredResourceUris []string    `json:"monitoredResourceUris"`
	SingleNssai           SingleNssai `json:"singleNassi,omitempty"`
	Dnn                   string      `json:"dnn,omitempty"`
	SubscriptionID        string      `json:"subscriptionId,omitempty"`
	PlmnID                PlmnID      `json:"plmnId,omitempty"`
}

type SdmSubsInfo struct {
	SupportFeatures     string `json:"supportFeatures,omitempty"`
	MtSmsSubscribed     bool   `json:"mtSmsSubScribed,omitempty"`
	MtSmsBarringAll     bool   `json:"mtSmsBarringAll,omitempty"`
	MtSmsBarringRoaming bool   `json:"mtSmsBarringRoaming,omitempty"`
	MoSmsSubscribed     bool   `json:"moSmsSubScribed,omitempty"`
	MoSmsBarringAll     bool   `json:"moSmsBarringAll,omitempty"`
	MoSmsBarringRoaming bool   `json:"moSmsBarringRoaming,omitempty"`
	SharedSmsMngDataIds string `json:"sharedSmsMngDataIds,omitempty"`
}

type SmsManagementSubscriptionData struct {
	SupportFeatures     string `json:"supportFeatures,omitempty"`
	MtSmsSubscribed     bool   `json:"mtSmsSubScribed,omitempty"`
	MtSmsBarringAll     bool   `json:"mtSmsBarringAll,omitempty"`
	MtSmsBarringRoaming bool   `json:"mtSmsBarringRoaming,omitempty"`
	MoSmsSubscribed     bool   `json:"moSmsSubScribed,omitempty"`
	MoSmsBarringAll     bool   `json:"moSmsBarringAll,omitempty"`
	MoSmsBarringRoaming bool   `json:"moSmsBarringRoaming,omitempty"`
}

type SubInfo struct {
	SupportFeatures     string `json:"supportFeatures,omitempty"`
	MtSmsSubscribed     bool   `json:"mtSmsSubScribed,omitempty"`
	MtSmsBarringAll     bool   `json:"mtSmsBarringAll,omitempty"`
	MtSmsBarringRoaming bool   `json:"mtSmsBarringRoaming,omitempty"`
	MoSmsSubscribed     bool   `json:"moSmsSubScribed,omitempty"`
	MoSmsBarringAll     bool   `json:"moSmsBarringAll,omitempty"`
	MoSmsBarringRoaming bool   `json:"moSmsBarringRoaming,omitempty"`
	SubscriptionID      string `json:"subscriptionId,omitempty"`
}

type ModificationNotification struct {
	NotifyItems []*NotifyItem `json:"notifyItems"`
}

type UECM struct {
	SmsfId   string
	MapId    string
	Mnc      string
	Mcc      string
	DiaName  string
	DiaRealm string
}

type UECMv1 struct {
	SmsfId string
}

type SDM struct {
	SmsfId  string
	SmsfUrl string
	Supi    string
}

func (s SDM) JsonMarshal() (body utypes.Any) {

	subNotiUrl := fmt.Sprintf("%s/nudm-svc/v2/sdm-change-notify/%s", s.SmsfUrl, s.Supi)

	body = SdmSubscribction{
		NfInstanceID:          s.SmsfId,
		ImplicitUnsubscribe:   true,
		CallbackReference:     subNotiUrl,
		MonitoredResourceUris: []string{subNotiUrl},
	}

	return body
}

func (e UECM) JsonMarshal() (body utypes.Any) {
	body = UECMReq{
		SmsfInstanceID: e.SmsfId,
		PlmnID: PlmnID{
			Mnc: e.Mnc,
			Mcc: e.Mcc,
		},
		SmsfMAPAddress: e.MapId,
		SmsfDiameterAddress: SmsfDiameterAddress{
			Name:  e.DiaName,
			Realm: e.DiaRealm,
		},
	}

	return body
}

func (e UECM) MakeJsonForm() (request UECMReq) {

	request = UECMReq{
		SmsfInstanceID: e.SmsfId,
		PlmnID: PlmnID{
			Mnc: e.Mnc,
			Mcc: e.Mcc,
		},
		SmsfMAPAddress: e.MapId,
		SmsfDiameterAddress: SmsfDiameterAddress{
			Name:  e.DiaName,
			Realm: e.DiaRealm,
		},
	}

	return request
}

func (e UECM) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	//	mutex.Lock()
	//	defer mutex.Unlock()
	request := UECMReq{
		SmsfInstanceID: e.SmsfId,
		PlmnID: PlmnID{
			Mnc: e.Mnc,
			Mcc: e.Mcc,
		},
		SmsfMAPAddress: e.MapId,
		SmsfDiameterAddress: SmsfDiameterAddress{
			Name:  e.DiaName,
			Realm: e.DiaRealm,
		},
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		return reqBody, err
	}

	return reqBody, err
}
func (s SDM) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//ar mutex = &sync.Mutex{}

	subNotiUrl := fmt.Sprintf("%s/nudm-svc/v2/sdm-change-notify/%s", s.SmsfUrl, s.Supi)
	//utex.Lock()
	//efer mutex.Unlock()
	request := SdmSubscribction{
		NfInstanceID:          s.SmsfId,
		ImplicitUnsubscribe:   true,
		CallbackReference:     subNotiUrl,
		MonitoredResourceUris: []string{subNotiUrl},
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		ulog.Info("json.Marshal Error, [%s]", err)
		return reqBody, err
		//		panic(err)
	}

	return reqBody, err
}

func (e UECMv1) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	//	mutex.Lock()
	//	defer mutex.Unlock()
	request := UECMReqv1{
		SmsfId: e.SmsfId,
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		return reqBody, err
	}

	return reqBody, err
}
