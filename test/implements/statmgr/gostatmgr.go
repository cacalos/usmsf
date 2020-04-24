package statmgr

/*
import (
	"time"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/uregi"
	"camel.uangel.com/ua5g/ulib.git/ustat"
	"github.com/prometheus/client_golang/prometheus"

	"uangel.com/usmsf/interfaces"
)

//StatMgr class
type GoStatMgr struct {
	mf             uregi.MetricFactory
	registry       ustat.Registry
	transcTimerMap map[string]ustat.TransactionTimer
	counterMap     map[string]ustat.Counter
	gaugeMap       map[string]ustat.Gauge
	meterMap       map[string]ustat.Meter
	histogramMap   map[string]prometheus.Histogram
	timerMap       map[string]ustat.Timer
}

var logger = ulog.GetLogger("com.uangel.usmsf.statmgr")

func NewGoStatMgr(cfg uconf.Config, mf uregi.MetricFactory, lf ulog.LoggerFactory) interfaces.StatMgr {
	mgr := &GoStatMgr{
		transcTimerMap: map[string]ustat.TransactionTimer{},
		counterMap:     map[string]ustat.Counter{},
		gaugeMap:       map[string]ustat.Gauge{},
		meterMap:       map[string]ustat.Meter{},
		histogramMap:   map[string]prometheus.Histogram{},
		timerMap:       map[string]ustat.Timer{},
	}
	mgr.mf = mf

	registry := cfg.GetString("statistic.registry", "default")
	mgr.registry = mf.Get(registry)

	items := cfg.GetConfig("statistic.items")
	if items == nil {
		logger.Warn("Fail to find config=statistics.items")
		return mgr
	}

	for _, name := range items.Keys() {
		loggers.InfoLogger().Comment("item=%+v", name)

		statType := items.GetString(name)

		switch statType {
		case "TransactionTimer":
			mgr.transcTimerMap[name] = ustat.GetOrRegisterTransactionTimer(name, mgr.registry)
		case "Counter":
			mgr.counterMap[name] = ustat.GetOrRegisterCounter(name, mgr.registry)
		case "Gauge":
			mgr.gaugeMap[name] = ustat.GetOrRegisterGauge(name, mgr.registry)
		case "Meter":
			mgr.meterMap[name] = ustat.GetOrRegisterMeter(name, mgr.registry)
		case "Histogram":
			histogram := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name, Buckets: prometheus.DefBuckets})
			mgr.registry.Register(name, ustat.NewHistogramFromPrometheus(histogram))
			mgr.histogramMap[name] = histogram
		case "Timer":
			mgr.timerMap[name] = ustat.GetOrRegisterTimer(name, mgr.registry)
		default:
			logger.Warn("Invalid statistic item=%s type=%s", name, statType)
		}
	}

	return mgr
}

type nilTimerContext struct {
}

func (r *nilTimerContext) End(statusCode int) {
	// do nothing
}

func (s *GoStatMgr) StartTranscTimer(item string) (ustat.TimerContext, error) {
	stat, ok := s.transcTimerMap[item]
	if !ok {
		return &nilTimerContext{}, errcode.SystemError("Failt to find statistic item=%s", item)
	}

	return stat.Time(), nil
}

func (s *GoStatMgr) IncCounter(item string) error {
	stat, ok := s.counterMap[item]
	if !ok {
		return errcode.SystemError("Failt to find statistic item=%s", item)
	}

	stat.Inc(1)
	return nil
}

func (s *GoStatMgr) UpdateGauge(item string, value int64) error {
	stat, ok := s.gaugeMap[item]
	if !ok {
		return errcode.SystemError("Failt to find statistic item=%s", item)
	}

	stat.Update(value)
	return nil
}

func (s *GoStatMgr) MarkMeter(item string) error {
	stat, ok := s.meterMap[item]
	if !ok {
		return errcode.SystemError("Failt to find statistic item=%s", item)
	}

	stat.Mark(1)
	return nil
}

func (s *GoStatMgr) UpdateHistogram(item string, startTime time.Time) error {
	stat, ok := s.histogramMap[item]
	if !ok {
		return errcode.SystemError("Failt to find statistic item=%s", item)
	}

	stat.Observe(float64(time.Now().Sub(startTime)) / float64(time.Second))
	return nil
}

func (s *GoStatMgr) UpdateTimer(item string, startTime time.Time) error {
	stat, ok := s.timerMap[item]
	if !ok {
		return errcode.SystemError("Failt to find statistic item=%s", item)
	}

	stat.UpdateSince(startTime)
	return nil
}
*/
