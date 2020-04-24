package main

import (
	/*
		"fmt"
		"io/ioutil"
		"net/http"
		"os"
		"sync/atomic"
		"time"

		"camel.uangel.com/ua5g/ulib.git/testhelper"
		"camel.uangel.com/ua5g/ulib.git/uconf"
		"camel.uangel.com/ua5g/ulib.git/utypes"
	*/
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"sync/atomic"
	"time"

	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/utypes"

	"camel.uangel.com/ua5g/usmsf.git/msg5g"

	jsoniter "github.com/json-iterator/go"
	"github.com/philippfranke/multipart-related/related"
)

const NFMoTesterName = "nfmo"

func init() {
	RegisterLoadTester(NFMoTesterName, &NFSmsMoTester{})
}

type NFSmsMoTester struct {
	params *Params

	config      uconf.Config
	metrics     *HTTPMetrics
	clients     []*HTTPClient
	clientCount int
	count       uint32
	token       string
	supiPfx     int

	smsRecordId string
	contentsId  string
	Gpsi        string
	AccessType  string

	supi      string
	perfonoff bool
}

func (t *NFSmsMoTester) Initialize(params *Params) error {
	var err error
	t.params = params
	cfg := testhelper.LoadConfigFromFile(params.Conf)
	if cfg == nil {
		return fmt.Errorf("Failed to load configuratoin (conf=%v)", params.Conf)
	}

	t.config = cfg

	//	t.token = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS8iLCJhdWQiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS9uc21zZi92Mi80NTAwNjEyMzQ1Njc4Iiwic3ViIjoidXNyXzEyMyIsInNjb3BlIjoicmVhZCB3cml0ZSIsImlhdCI6MTQ1ODc4NTc5NiwiZXhwIjoxNjY4ODcyMTk2fQ.ePtZCfzIMNaeRCV1O5EtNMQ0myMBVffM9z95e4p9u24"

	t.supiPfx = cfg.GetInt(NFMoTesterName+".supi-prefix", 45000)
	t.smsRecordId = cfg.GetString(NFMoTesterName+".smsRecordId", "abcdefg-123")
	t.contentsId = cfg.GetString(NFMoTesterName+".contentsId", "abcdefg-123@amf.com")
	t.Gpsi = cfg.GetString(NFMoTesterName+".Gpsi", "msisdn-01033334444")
	t.AccessType = cfg.GetString(NFMoTesterName+".AccessType", "3GPP")
	t.perfonoff = cfg.GetBoolean(NFActTesterName+".perfon", false)
	t.supi = t.params.supi

	t.metrics = NewHTTPMetrics()
	t.clientCount = params.ClientCount
	t.clients = make([]*HTTPClient, t.clientCount, t.clientCount)
	for i := 0; i < t.clientCount; i++ {
		t.clients[i], err = NewHTTPClientWithConfig(cfg, NFMoTesterName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *NFSmsMoTester) MakeMo() ([]byte, error) {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var b bytes.Buffer
	var err error

	//	smsRecordId := "abcdefg-123"
	//	contentsId := "abcdefg-123@amf.com"

	request := msg5g.UplinkSMS{
		SmsRecordID: t.smsRecordId,
		SmsPayloads: []msg5g.RefToBinaryData{
			{ContentID: t.contentsId},
		},
		Gpsi:       t.Gpsi,
		AccessType: t.AccessType,
	}

	body, err := json.Marshal(request)

	w := related.NewWriter(&b)
	w.SetBoundary("Boundary")

	rootPart, err := w.CreateRoot("", "application/json", nil)
	if err != nil {
		fmt.Errorf("CreateRoot fail:%s", err)
		return nil, err
	}
	rootPart.Write(body)

	header := make(textproto.MIMEHeader)

	header.Set("Content-Type", "application/vnd.3gpp.sms")
	header.Set("Content-ID", t.contentsId)
	nextPart, err := w.CreatePart("", header)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var cpdata []byte
	cpdata = make([]byte, 100, 100)
	cpdata = []byte{0x09, 0x01, 0x1F, 0x00, 0x02, 0x00, 0x07, 0x91, 0x28, 0x01, 0x92, 0x99, 0x11, 0x41, 0x13, 0x01, 0x04, 0x0B, 0x81, 0x10, 0x70, 0x00, 0x00, 0x10, 0xF2, 0x00, 0x00, 0x06, 0x47, 0x34, 0xDA, 0x6E, 0xB7, 0x03}

	nextPart.Write(cpdata)

	if err := w.Close(); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return b.Bytes(), err
}

func (t *NFSmsMoTester) Execute() error {

	var supi string

	if t.perfonoff == true {
		idx := atomic.AddUint32(&t.count, 1)

		supi = fmt.Sprintf("imsi-%d%010d", t.supiPfx, idx)
	} else {
		supi = "imsi-" + t.supi

	}

	fmt.Println(supi, len(supi))

	if len(supi) != 20 {

		fmt.Println("Invalid SUPI : %d, SUPI_LEN : %s", supi, len(supi))
		return errors.New("Invalid supi len")
	}

	path := fmt.Sprintf("/nsmsf-sms/v1/ue-contexts/%s/sendsms", supi)
	contentsType := "multipart/related;boundary=Boundary"

	hdrs := utypes.Labels{}
	hdrs["accept"] = "application/json"
	hdrs["Content-Type"] = contentsType
	//	hdrs["Authorization"] = t.token

	body, err := t.MakeMo()

	if err != nil {
		fmt.Println(err)
		return err
	}

	now := t.metrics.Start()
	//	client := t.clients[int(t.count)%t.clientCount]
	rsp, err := t.clients[int(t.count)%t.clientCount].SendPerfVerbose(t.metrics, t.params.Verbose, http.MethodPost, path, hdrs, body)
	if err != nil {
		//		if t.params.Verbose {
		fmt.Printf("Failed to send Uplink(MO), supi:%s, err:%v", supi, err)
		return err

	}

	//	t.metrics.Stop(now, rsp.StatusCode)
	t.metrics.UplinkStop(now, rsp.StatusCode)

	return nil
}

func (t *NFSmsMoTester) Report(isFinal bool) {
	fmt.Printf("[%v]\n", time.Now())
	fmt.Printf("NFManagement.Uplink(MO)\n")
	t.metrics.Report(true, true, isFinal)
	fmt.Printf("\n")
}

func (t *NFSmsMoTester) Finalize() {
	fmt.Printf("[%v] FINAL --------------------------------------------\n", time.Now())
	t.metrics.Report(true, true, true)
	fmt.Printf("\n")
}
