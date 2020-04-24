package tcpmgr

import (
	"net"
)

const (
	MO_MSG  = 1
	MO_RESP = 2
	MT_MSG  = 3
	MT_RESP = 4
	MT_ERR  = 5
)

type MoData struct {
	TotalLen int

	Type byte

	SupiLen byte
	Supi    [128]byte

	GpsiLen byte
	Gpsi    [128]byte

	ContentDataLen byte
	ContentData    [248]byte
	InterFInfo     InterFInfo_t
	CommonConf     CommonConf_t
}

type MtData struct {
	TotalLen int

	Type byte

	MsgTypeLen int
	MsgType    [32]byte

	ResultCodeLen int
	ResultCode    [64]byte

	// for resp
	Diag_id_len int
	Diag_id     [32]byte

	Acn     int
	Prov_id int
	Inv_id  int
	Hop_id  int
	End_id  int
	Peer_id int

	Orig_realm_len int
	Orig_realm     [24]byte

	Orig_host_len int
	Orig_host     [24]byte

	Smsc_node_len int
	Smsc_node     [24]byte

	Session_id_len int
	Session_id     [512]byte

	//	CcLen     int
	CauseCode int

	SupiLen int
	Supi    [128]byte

	GpsiLen int
	Gpsi    [128]byte

	MmsLen byte
	Mms    byte

	ContentDataLen int
	ContentData    [248]byte

	InterFInfo InterFInfo_t
	CommonConf CommonConf_t

	/** For Cdr **/
	Result byte
}

type Redis_Response struct {
	// for resp
	Diag_id string `json:"diag_id,omitempty"`

	Acn     int `json:"acn,omitempty"`
	Prov_id int `json:"prov_id,omitempty"`
	Inv_id  int `json:"inv_id,omitempty"`
	Hop_id  int `json:"hop_id,omitempty"`
	End_id  int `json:"end_id,omitempty"`
	Peer_id int `json:"peer_id,omitempty"`

	Orig_realm string `json:"orig_ream,omitempty"`
	Orig_host  string `json:"orig_host,omitempty"`
	Smsc_node  string `json:"smsc_node,omitempty"`
	Session_id string `json:"session_id,omitempty"`
}

type CommonConf_t struct {
	PlmnIdLen              int
	PlmnId                 int
	SmsfInstanceIdLen      int
	SmsfInstanceId         string
	SmsfMapAddressLen      int
	SmsfMapAddress         string
	SmsfDiameterAddressLen int
	SmsfDiameterAddress    string
	SmsfPointCodeLen       int
	SmsfPointCode          int
	SmsfSsnLen             int
	SmsfSsn                int
}

type InterFInfo_t struct {
	NameLen      int
	Name         string
	IsdnLen      int
	Isdn         string
	PcLen        int
	Pc           int
	SsnLen       int
	Ssn          int
	TypeLen      int
	Type         int
	FlowCtrlLen  int
	FlowCtrl     int
	DestHostLen  int
	DestHost     string
	DestRealmLen int
	DestRealm    string
	DescLen      int
	Desc         string
}

type HttpIfMoMsg struct {
	ContetnsId string `json:"ContentsId"`
	Gpsi       string `json:"Gpsi"`
	RpData     []byte `json:"rpdata"`
}

type HttpIfMtMsg struct {
	ContetnsId string `json:"ContentsId"`
	Mms        bool   `json:"mms"`
	RpData     []byte `json:"RpData"`
}

type MtFailNoti struct {
	MsgType    string
	ResultCode string
}

///////////////////
var Cli CliConnInfo
var MsgProxyCli CliConnInfo

type CliConnInfo struct {
	Client net.Conn
	err    error
}

const TRACE_REG_MAX = 10

const (
	DEFAULT_ADDR_LEN = 24
	DEFAULT_DIA_HOST = 64
)

const (
	SmscNodeStoragePopFail = 1
	CommonStoragePopFail   = 2
)

type TraceInfo struct {
	Target   string `json:"target,omitempty"`
	Level    int    `json:"level,omitempty"`
	Duration int64  `json:"duration,omitempty"`
	//	Create_UnixTime time.Time `json:"create_unixtime"`
	Create_UnixTime int64 `json:"create_unixtime,omitempty"`
}
