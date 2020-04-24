package svctracemgr

import (
	"fmt"
	"time"
)

type NFTrace struct {
	trace *TraceSvcPod
}

func NewTraceWrite(trace *TraceSvcPod) *NFTrace {
	go func() {
		for {
			now := time.Now()
			trace.FileName = fmt.Sprintf("%d%02d%02d", now.Year(), now.Month(), now.Day())
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		idx := 0

		for {
			now := int64(time.Now().Unix())
			if trace.traceRegFlag[idx] == true {
				if trace.traceDuration[idx] < now {
					trace.traceRegFlag[idx] = false
					trace.traceDuration[idx] = 0
					trace.trace[idx].Target = ""
					trace.trace[idx].Level = 0
					trace.trace[idx].Duration = -1
					trace.trace[idx].Create_UnixTime = -1
					trace.RegCount--
				}
			}

			idx++
			if idx >= TRACE_REG_MAX {
				idx = 0
				time.Sleep(1 * time.Second)
			}
		}
	}()

	s := &NFTrace{
		trace: trace,
	}

	return s
}
