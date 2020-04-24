package msg5g

import (
	"bytes"
	"fmt"
	"net/textproto"

	//"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/philippfranke/multipart-related/related"

	"camel.uangel.com/ua5g/ulib.git/ulog"
)

type N1N2Request struct {
	N1MessageContainer N1MessageContainer `json:"n1MessageContainer,omitempty"`
	//	N2InfoContainer        N2InfoContainer    `json:"n2InfoContainer,omitempty"`
	SkipInd           bool   `json:"skipInd,omitempty"`
	LastMsgIndication bool   `json:"lastMsgIndication,omitempty"`
	PduSessionId      int    `json:"pduSessionId,omitempty"`
	LcsCorrelationId  string `json:"lcsCorrelationId,omitempty"`
	Ppi               int    `json:"ppi,omitempty"`
	//	Arp                    Arp                `json:"arp,omitempty"`
	//	Qi                     int                `json:"5qi,omitempty"`
	N1n2FailureTxfNotifURI string `json:"n1n2FailureTxtNotifURI,omitempty"`
	SmfReallocationInd     bool   `json:"smfReallocationInd,omitempty"`
	//	AreaOfValidity         AreaOfValidity     `json:"areaOfValidity,omitempty"`
	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type N1N2MsgTxfrFailureNotification struct {
	Cause          string `json:"cause"`
	N1n2MsgDataUri string `json:"n1n2MsgDataUri"`
}

type UeReachableReq struct {
	Reachability      string `json:"reachability"`
	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type N1N2_RESP struct {
	Cpmsg    []byte
	MmsFlag  bool
	Boundary string
}

type N1N2_MT struct {
	Cpmsg    []byte
	MmsFlag  bool
	SmsfUrl  string
	Supi     string
	Boundary string
}

type UeReach struct {
	ReachInd string
}

const (
	UNREACHABLE     = "UNREACHABLE"
	REACHABLE       = "REACHABLE"
	REGULATORY_ONLY = "REGULATORY_ONLY"
)

func (n N1N2_MT) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//var mutex = &sync.Mutex{}

	contentsId := fmt.Sprintf("%s@smsf.com", RandASCIIBytes(10))

	failNotiUrl := fmt.Sprintf("%s/namf-svc/v1/sms-failure-notify/%s", n.SmsfUrl, n.Supi)

	//mutex.Lock()
	//defer mutex.Unlock()
	request := N1N2Request{
		N1MessageContainer: N1MessageContainer{
			N1MessageClass: "SMS",
			N1MessageContent: RefToBinaryData{
				ContentID: contentsId,
			},
		},
		LastMsgIndication:      n.MmsFlag,
		N1n2FailureTxfNotifURI: failNotiUrl,
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		ulog.Error("json.Marshal Error, [%s]", err)
		return reqBody, err
	}

	var b bytes.Buffer
	w := related.NewWriter(&b)
	w.SetBoundary(n.Boundary)

	rootPart, err := w.CreateRoot("", "application/json", nil)
	if err != nil {
		ulog.Error("CreatRoot, [%s]", err)

		return reqBody, err
	}
	rootPart.Write(reqBody)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/vnd.3gpp.5gnas")
	header.Set("Content-ID", contentsId)

	nextPart, err := w.CreatePart("", header)
	if err != nil {
		return reqBody, err
	}

	nextPart.Write(n.Cpmsg)

	if err := w.Close(); err != nil {

		return reqBody, err
	}

	return b.Bytes(), err
}

func (n N1N2_RESP) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	contentsId := fmt.Sprintf("%s@smsf.com", RandASCIIBytes(10))

	//	mutex.Lock()
	//	defer mutex.Unlock()
	request := N1N2Request{
		N1MessageContainer: N1MessageContainer{
			N1MessageClass: "SMS",
			N1MessageContent: RefToBinaryData{
				ContentID: contentsId,
			},
		},
		LastMsgIndication: n.MmsFlag,
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		ulog.Error("json.Marshal Error, [%s]", err)
		return reqBody, err
	}

	var b bytes.Buffer
	w := related.NewWriter(&b)
	w.SetBoundary(n.Boundary)

	rootPart, err := w.CreateRoot("", "application/json", nil)
	if err != nil {
		ulog.Error("CreatRoot, [%s]", err)
		return reqBody, err

	}
	rootPart.Write(reqBody)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/vnd.3gpp.5gnas")
	header.Set("Content-ID", contentsId)

	nextPart, err := w.CreatePart("", header)
	if err != nil {
		return reqBody, err
	}

	nextPart.Write(n.Cpmsg)

	if err := w.Close(); err != nil {
		return reqBody, err
	}

	return b.Bytes(), err
}

func (u UeReach) Make() (reqBody []byte, err error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var mutex = &sync.Mutex{}

	//	mutex.Lock()
	//	defer mutex.Unlock()

	request := UeReachableReq{
		Reachability: u.ReachInd,
	}

	reqBody, err = json.Marshal(request)

	if err != nil {
		return reqBody, err
	}

	return reqBody, err
}
