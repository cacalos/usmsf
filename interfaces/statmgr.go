package interfaces

import (
	"net/http"
	"time"
	
	"camel.uangel.com/ua5g/ulib.git/ustat"
)

const (
	StatDBNotFound     = 550
	StatDBAlreadyExist = 551
)

var statText = map[int]string{
	StatDBNotFound:     "StatDBNotFound",
	StatDBAlreadyExist: "StatDBAlreadyExist",
}

func StatText(code int) string {
	text := http.StatusText(code)
	if text != "" {
		return text
	}

	return statText[code]
}

type StatMgr interface {
	StartTranscTimer(item string) (ustat.TimerContext, error)
	IncCounter(item string) error
	UpdateGauge(item string, value int64) error
	MarkMeter(item string) error
	UpdateHistogram(item string, startTime time.Time) error
	UpdateTimer(item string, startTime time.Time) error
}
