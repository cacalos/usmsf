package tcptracemgr

import ()

const TRACE_REG_MAX = 10

type TraceInfo struct {
	Target          string `json:"target,omitempty"`
	Level           int    `json:"level,omitempty"`
	Duration        int64  `json:"duration,omitempty"`
	Create_UnixTime int64  `json:"create_unixtime,omitempty"`
}
