package main

import (
	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/utypes"

	"fmt"

	jsoniter "github.com/json-iterator/go"

	//	"github.com/go-openapi/strfmt"
	//	"github.com/google/uuid"
	//	"github.com/jinzhu/copier"
	//"io/util"
	"net/http"
	//"os"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
	//	"sync"
	//"errors"
	"sync/atomic"
	"time"
)

const NFActTesterName = "nfact"

func init() {
	RegisterLoadTester(NFActTesterName, &NFActivatorTester{})
}

type NFActivatorTester struct {
	params *Params

	config      uconf.Config
	metrics     *HTTPMetrics
	clients     []*HTTPClient
	clientCount int
	count       uint32
	idx         uint32

	supiPfx          int
	Gpsi             string
	AccessType       string
	AmfId            string
	UdmGroupId       string
	RoutingIndicator string
	supi             string
	perfonoff        bool
}

func (t *NFActivatorTester) Initialize(params *Params) error { // 여기에 커넥션 갯수랑 기타.. config 파일들 들어가겠지?
	t.params = params
	var err error

	cfg := testhelper.LoadConfigFromFile(params.Conf)

	if cfg == nil {
		return fmt.Errorf("Failed to load configuratoin (conf=%v)", params.Conf)
	}

	t.config = cfg
	t.metrics = NewHTTPMetrics()
	t.supiPfx = cfg.GetInt(NFActTesterName+".supi-prefix", 43000)
	t.Gpsi = cfg.GetString(NFActTesterName+".Gpsi", "msisdn-01040001234")
	t.AccessType = cfg.GetString(NFActTesterName+".AccessType", "3GPP_ACCESS")
	t.AmfId = cfg.GetString(NFActTesterName+".AmfId", "1234-b32311-737c123-1876abcd7")
	t.UdmGroupId = cfg.GetString(NFActTesterName+".UdmGroupId", "g1")
	t.RoutingIndicator = cfg.GetString(NFActTesterName+".RoutingIndicator", "3")
	t.perfonoff = cfg.GetBoolean(NFActTesterName+".perfon", false)
	t.supi = t.params.supi

	t.clientCount = params.ClientCount
	t.clients = make([]*HTTPClient, t.clientCount, t.clientCount)
	for i := 0; i < t.clientCount; i++ {
		t.clients[i], err = NewHTTPClientWithConfig(cfg, NFActTesterName)
		if err != nil {
			return err
		}
	}

	return nil

}

func (t *NFActivatorTester) Execute() error {

	//	var mutex = &sync.Mutex{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	//	var convSupi string
	var smsfURL string
	var reqbody []byte
	var err error
	var supi string

	token := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS8iLCJhdWQiOiJodHRwczovL3Ntc2YudWFuZ2VsLmNvbS9uc21zZi92Mi80NTAwNjEyMzQ1Njc4Iiwic3ViIjoidXNyXzEyMyIsInNjb3BlIjoicmVhZCB3cml0ZSIsImlhdCI6MTQ1ODc4NTc5NiwiZXhwIjoxNjY4ODcyMTk2fQ.ePtZCfzIMNaeRCV1O5EtNMQ0myMBVffM9z95e4p9u24"

	contentsType := "application/json"

	hdrs := utypes.Labels{}

	hdrs["accept"] = contentsType
	hdrs["Content-Type"] = contentsType
	hdrs["Authorization"] = token

	//	count := atomic.AddUint32(&t.idx, 1)
	//	mutex.Lock()
	//	supi := fmt.Sprintf("imsi-%d%010d", t.supiPfx, count)
	//	mutex.Unlock()

	if t.perfonoff == false {
		supi = "imsi-" + t.supi
	} else {
		count := atomic.AddUint32(&t.idx, 1)
		supi = fmt.Sprintf("imsi-%d%010d", t.supiPfx, count)

	}

	fmt.Println(supi, len(supi))

	if len(supi) != 20 {

		fmt.Printf("INVALID SUPI LEN(supi:%s, %d)", supi, len(supi))
		return err
	}

	smsfURL = fmt.Sprintf("/nsmsf-sms/v1/ue-contexts/%s", supi)

	request := msg5g.UeSmsContextData{
		Supi:             supi,
		Gpsi:             t.Gpsi,
		AccessType:       t.AccessType,
		AmfId:            t.AmfId,
		UdmGroupId:       t.UdmGroupId,
		RoutingIndicator: t.RoutingIndicator,
	}

	reqbody, err = json.Marshal(request)

	now := t.metrics.Start()
	resp, err := t.clients[(int(t.idx)%t.clientCount)].SendPerfVerbose(t.metrics, t.params.Verbose, http.MethodPut, smsfURL, hdrs, reqbody)
	if err != nil {
		fmt.Println("client.Call Error : ", err)
		return err
	}

	t.metrics.ActivateStop(now, resp.StatusCode)

	return nil
}

func (t *NFActivatorTester) Report(isFinal bool) {
	fmt.Printf("[%v]\n", time.Now())
	//	fmt.Printf("Management: NFSubscriptionUpdate, NFUpdate, NFStatusNotify\n")
	t.metrics.Report(true, true, isFinal)
	fmt.Printf("\n")

}

func (t *NFActivatorTester) Finalize() {
	fmt.Printf("[%v] FINAL --------------------------------------------\n", time.Now())
	t.metrics.Report(true, true, true)
	fmt.Printf("\n")
}
