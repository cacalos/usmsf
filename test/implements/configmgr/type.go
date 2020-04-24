package configmgr

// To to get configFile
const service_name string = "SMSF"
const common_config_name string = "SmsfConfiguration"
const smscinfo_config_name string = "SmscConfiguration"
const decision_config_name string = "DecisionConfiguration"

const (
	decision = iota
	smsf
	smsc
)

// Map Data
var commonData = CommonConfiguration{}
var nodeMap = make(map[string]SmscNode)
var prefixMap = make(map[string]SmscPrefix)

var decisionMap = make(map[string]DecisionConf)
var (
	configMap = make(map[int]string)
	count     = 0
)
var (
	watchIdMap = make(map[int]string)
	watchcnt   = 0
)

const server_port string = "8090"

type NotifyConfigRespData struct {
	ConfId string `json:"conf_id"`
}

type DecisionConfigRespData struct {
	ConfId string `json:"conf_id"`
	MetaId string `json:"meta_id"`
	Tag    string `json:"tag"`

	Configuration DecisionTableInfo `json:"configuration"`
}

type DecisionTableInfo struct {
	Decision []DecisionConf `json:"DECISION"`
}

type DecisionConf struct {
	Prefix   string `json:"Prefix"`
	Decision string `json:"Decision"`
}

type ConfigCtrlReqData struct {
	ConfId string `json:"conf_id"`
}

type CommonConfigRespData struct {
	ConfId        string              `json:"conf_id"`
	MetaId        string              `json:"meta_id"`
	Tag           string              `json:"tag"`
	Configuration CommonConfiguration `json:"configuration"`
}

type CommonConfiguration struct {
	PlmnId              int    `json:"PLMN ID"`
	SmsfInstanceId      string `json:"SMSF InstanceId"`
	SmsfMapAddress      string `json:"SMSF-MAP-Address"`
	SmsfDiameterAddress string `json:"SMSF-Diameter-Address"`
	SmsfPointCode       int    `json:"SMSF-Point-Code"`
	SmsfSsn             int    `json:"SMSF-SSN"`
}

type SmscConfigRespData struct {
	ConfId        string        `json:"conf_id"`
	MetaId        string        `json:"meta_id"`
	Tag           string        `json:"tag"`
	Configuration SmscTableInfo `json:"configuration"`
}

type SmscTableInfo struct {
	Node   []SmscNode   `json:"NODE"`
	Prefix []SmscPrefix `json:"PREFIX"`
}

type SmscNode struct {
	//NODE
	Name       string `json:"Name"`
	Isdn       string `json:"Isdn"`
	Pc         int    `json:"Pc"`
	Ssn        int    `json:"Ssn"`
	Type       int    `json:"Type"`
	FlowCtrl   int    `json:"FlowCTRL"`
	Dest_host  string `json:"Dest_host"`
	Dest_realm string `json:"Dest_realm"`
	Desc       string `json:"Desc"`
}

type SmscPrefix struct {
	//PREFIX
	Prefix   string `json:"Prefix"`
	SmscName string `json:"SMSC Name"`
}

// REST API Watch Struct(Req + Resp)
type WatchCtrlReqData struct {
	//	WatchId  string `json:"watch_id"`
	ConfId   string `json:"conf_id"`
	CallBack string `json:"call_back"`
}

type WatchCtrlResData struct {
	ConfId   string `json:"conf_id"`
	CallBack string `json:"call_back"`
	WatchId  string `json:"watch_id"`
}

type WatchDelete struct {
	WatchId string `json:"watch_id"`
}
