package msg5g

import (
	"camel.uangel.com/ua5g/ulib.git/ulog"
	jsoniter "github.com/json-iterator/go"
)

type UeSmsContextData struct {
	Supi             string        `json:"supi"`
	Pei              string        `json:"pei,omitempty"`
	AmfId            string        `json:"amfId"`
	Guamis           []Guami       `json:"guamis,omitempty"`
	AccessType       string        `json:"accessType,omitempty"`
	Gpsi             string        `json:"gpsi,omitempty"`
	UeLocation       UserLocation  `json:"ueLocation,omitempty"`
	UeTimeZone       string        `json:"ueTimeZone,omitempty"`
	TraceData        TraceData     `json:"traceData,omitempty"`
	BackupAmfInfo    BackupAmfInfo `json:"backupAmfInfo,omitempty"`
	UdmGroupId       string        `json:"udmGroupId,omitempty"`
	RoutingIndicator string        `json:"routingIndicator,omitempty"`
}

type UplinkSMS struct {
	SmsRecordID string            `json:"smsRecordId"`
	SmsPayloads []RefToBinaryData `json:"smsPayloads"`
	AccessType  string            `json:"accessType,omitempty"`
	Gpsi        string            `json:"gpsi,omitempty"`
	Pei         string            `json:"pei,omitempty"`
	UeLocation  UserLocation      `json:"ueLocation,omitempty"`
	UeTimneZone string            `json:"ueTimeZone,omitempty"`
}

type UplinkResp struct {
	SmsRecordID    string `json:"smsRecordId"`
	DeliveryStatus string `json:"deliveryStatus"`
}

type UpResp struct {
	RecordId string
}

func (u UpResp) Make() (respBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	//	mutex.Lock()
	//	defer mutex.Unlock()
	resp := UplinkResp{
		SmsRecordID:    u.RecordId,
		DeliveryStatus: SMS_DELIVERY_PENDING,
	}

	respBody, err = json.Marshal(resp)

	if err != nil {
		ulog.Info("json.Marshal Error, [%s]", err)
		return respBody, err
	}

	return respBody, err
}
