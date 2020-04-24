package controller

import (
	"net/http"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/uregi"
	"camel.uangel.com/ua5g/ulib.git/ustat"
)

// Stats forwader statistics
type Stats struct {
	mf               uregi.MetricFactory      // Metrics Factory
	registry         ustat.Registry           // statistics registry
	countersErr      []ustat.Counter          // counter statistics
	countersOper     []ustat.Counter          // counter statistics
	countersAct      []ustat.Counter          // counter statistics
	countersDeact    []ustat.Counter          // counter statistics
	countersUplink   []ustat.Counter          // counter statistics
	countersReg      []ustat.Counter          // counter statistics
	countersDereg    []ustat.Counter          // counter statistics
	countersSdmget   []ustat.Counter          // counter statistics
	countersReach    []ustat.Counter          // counter statistics
	countersN1n2     []ustat.Counter          // counter statistics
	countersSubs     []ustat.Counter          // counter statistics
	countersFailNoti []ustat.Counter          // counter statistics
	countersHttpif   []ustat.Counter          // counter statistics
	transcs          []ustat.TransactionTimer // transaction timer statistics
}

type ErrStatCounterItem int

const (
	InvalidHttpMsg = iota
	InvalidToken
	InvalidSDMGETSendHttpMsg
	InvalidSDMSUBSSendHttpMsg
	InvalidUECMSendHttpMsg
	InvalidN1N2SendHttpMsg
	InvalidReachSendHttpMsg
	ErrStatCounterItemMax
)

// statCounterNames forwarder counter statistics names
var ErrStatCounterNames = [...]string{
	"InvalidHttpMsg",
	"InvalidToken",
	"InvalidSDMGETSendHttpMsg",
	"InvalidSDMSUBSSendHttpMsg",
	"InvalidUECMSendHttpMsg",
	"InvalidN1N2SendHttpMsg",
	"InvalidReachSendHttpMsg",

	"unknown_err_counter",
}

// StatCounterItem forwarder counter statistic item
type StatCounterItem int

const (
	ActivateStat = iota
	DeactivateStat
	UplinkStat
	RegStat
	DeregStat
	SdmgetStat
	ReachStat
	N1n2Stat
	SubsStat

	//StatCounterItemMax forwarder counter statistic item max
	OperStatCounterItemMax
)

// statCounterNames forwarder counter statistics names
var statCounterNames = [...]string{
	"Activate",
	"Deactivate",
	"Uplink",
	"Registration",
	"Deregistration",
	"Sdmget",
	"Reachablility",
	"N1N2",
	"Subscription",

	"unknown_operation_counter",
}

type ActivateStatCounterItem int

const (
	ActivateTotal = iota
	Activate200
	Activate201
	Activate202
	Activate204
	Activate400
	Activate403
	Activate404
	Activate501
	Activate503
	Activate504
	ActivateEtcResp

	ActivateStatCounterItemMax
)

// statCounterNames forwarder counter statistics names
var ActivateStatCounterNames = [...]string{
	"ActivateTotal",
	"Act200",
	"Act201",
	"Act202",
	"Act204",
	"Act400",
	"Act403",
	"Act404",
	"Act501",
	"Act503",
	"Act504",
	"ActEtcResp",

	"Act_unknown_code_counter",
}

type DeactivateStatCounterItem int

const (
	DeactivateTotal = iota
	Deactivate200
	Deactivate201
	Deactivate202
	Deactivate204
	Deactivate400
	Deactivate403
	Deactivate404
	Deactivate501
	Deactivate503
	Deactivate504
	DeactivateEtcResp

	DeactivateStatCounterItemMax
)

var DeactivateStatCounterNames = [...]string{
	"DeactivateTotal",
	"Deact200",
	"Deact201",
	"Deact202",
	"Deact204",
	"Deact400",
	"Deact403",
	"Deact404",
	"Deact501",
	"Deact503",
	"Deact504",
	"DeactEtcResp",

	"Deact_unknown_code_counter",
}

type UplinkStatCounterItem int

const (
	UplinkTotal = iota
	Uplink200
	Uplink201
	Uplink202
	Uplink204
	Uplink400
	Uplink403
	Uplink404
	Uplink501
	Uplink503
	Uplink504
	UplinkEtcResp

	UplinkStatCounterItemMax
)

var UplinkStatCounterNames = [...]string{
	"UplinkTotal",
	"Uplink200",
	"Uplink201",
	"Uplink202",
	"Uplink204",
	"Uplink400",
	"Uplink403",
	"Uplink404",
	"Uplink501",
	"Uplink503",
	"Uplink504",
	"UplinkEtcResp",

	"Uplink_unknown_code_counter",
}

type RegStatCounterItem int

const (
	RegTotal = iota
	Reg200
	Reg201
	Reg202
	Reg204
	Reg400
	Reg403
	Reg404
	Reg501
	Reg503
	Reg504
	RegEtcResp

	RegStatCounterItemMax
)

var RegStatCounterNames = [...]string{
	"RegTotal",
	"Reg200",
	"Reg201",
	"Reg202",
	"Reg204",
	"Reg400",
	"Reg403",
	"Reg404",
	"Reg501",
	"Reg503",
	"Reg504",
	"RegEtcResp",

	"Reg_unknown_code_counter",
}

type DeregStatCounterItem int

const (
	DeregTotal = iota
	Dereg200
	Dereg201
	Dereg202
	Dereg204
	Dereg400
	Dereg403
	Dereg404
	Dereg501
	Dereg503
	Dereg504
	DeregEtcResp

	DeregStatCounterItemMax
)

var DeregStatCounterNames = [...]string{
	"DeregTotal",
	"Dereg200",
	"Dereg201",
	"Dereg202",
	"Dereg204",
	"Dereg400",
	"Dereg403",
	"Dereg404",
	"Dereg501",
	"Dereg503",
	"Dereg504",
	"DeregEtcResp",

	"Dereg_unknown_code_counter",
}

type SdmgetStatCounterItem int

const (
	SdmgetTotal = iota
	Sdmget200
	Sdmget201
	Sdmget202
	Sdmget204
	Sdmget400
	Sdmget403
	Sdmget404
	Sdmget501
	Sdmget503
	Sdmget504
	SdmgetEtcResp

	SdmgetStatCounterItemMax
)

var SdmgetStatCounterNames = [...]string{
	"SdmgetTotal",
	"Sdmget200",
	"Sdmget201",
	"Sdmget202",
	"Sdmget204",
	"Sdmget400",
	"Sdmget403",
	"Sdmget404",
	"Sdmget501",
	"Sdmget503",
	"Sdmget504",
	"SdmgetEtcResp",

	"Sdmget_unknown_code_counter",
}

type ReachStatCounterItem int

const (
	ReachTotal = iota
	Reach200
	Reach201
	Reach202
	Reach204
	Reach400
	Reach403
	Reach404
	Reach501
	Reach503
	Reach504
	ReachEtcResp

	ReachStatCounterItemMax
)

var ReachStatCounterNames = [...]string{
	"ReachTotal",
	"Reach200",
	"Reach201",
	"Reach202",
	"Reach204",
	"Reach400",
	"Reach403",
	"Reach404",
	"Reach501",
	"Reach503",
	"Reach504",
	"ReachEtcResp",

	"Reach_unknown_code_counter",
}

type N1n2StatCounterItem int

const (
	N1n2Total = iota
	N1n2200
	N1n2201
	N1n2202
	N1n2204
	N1n2400
	N1n2403
	N1n2404
	N1n2501
	N1n2503
	N1n2504
	N1n2EtcResp

	N1n2StatCounterItemMax
)

var N1n2StatCounterNames = [...]string{
	"N1n2Total",
	"N1n2200",
	"N1n2201",
	"N1n2202",
	"N1n2204",
	"N1n2400",
	"N1n2403",
	"N1n2404",
	"N1n2501",
	"N1n2503",
	"N1n2504",
	"N1n2EtcResp",

	"N1n2_unknown_code_counter",
}

type SubsStatCounterItem int

const (
	SubsTotal = iota
	Subs200
	Subs201
	Subs202
	Subs204
	Subs400
	Subs403
	Subs404
	Subs501
	Subs503
	Subs504
	SubsEtcResp

	SubsStatCounterItemMax
)

var SubsStatCounterNames = [...]string{
	"SubsTotal",
	"Subs200",
	"Subs201",
	"Subs202",
	"Subs204",
	"Subs400",
	"Subs403",
	"Subs404",
	"Subs501",
	"Subs503",
	"Subs504",
	"SubsEtcResp",

	"Subs_unknown_code_counter",
}

type HttpifStatCounterItem int

const (
	HttpifTotal = iota
	MoResp

	HttpifStatCounterItemMax
)

var HttpifStatCounterNames = [...]string{
	"HttpifTotal",
	"MoResp",

	"Httpif_unknown_code_counter",
}

////////////////////////////////////////////////////////////////////////////////
// Transaction Timer Utility functions
////////////////////////////////////////////////////////////////////////////////

//EndTransacTimer 전달된 error 값을 기반으로 Transaction Timer 통계 수집 후 종료한다.
func EndTransacTimer(timer ustat.TimerContext, err *error) {
	if err == nil || (*err) == nil {
		timer.End(http.StatusOK)
	} else {
		timer.End(errcode.GetCode(*err, http.StatusInternalServerError))
	}
}

//EndTransacTimerWitCode 전달된 code 값을 기반으로 Transaction Timer 통계 수집 후 종료한다.
func EndTransacTimerWitCode(timer ustat.TimerContext, code *int) {
	if code == nil || *code == 0 {
		timer.End(http.StatusOK)
	} else {
		timer.End(*code)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Stats functions
////////////////////////////////////////////////////////////////////////////////

// NewStats forwarder용 Statistics를 생성한다.
func NewStats(cfg uconf.Config, mf uregi.MetricFactory) *Stats {
	regname := cfg.GetString("statistic.registry", "default")

	s := &Stats{
		mf:             mf,
		registry:       mf.Get(regname),
		countersErr:    make([]ustat.Counter, ErrStatCounterItemMax),
		countersOper:   make([]ustat.Counter, OperStatCounterItemMax),
		countersAct:    make([]ustat.Counter, ActivateStatCounterItemMax),
		countersDeact:  make([]ustat.Counter, DeactivateStatCounterItemMax),
		countersUplink: make([]ustat.Counter, UplinkStatCounterItemMax),
		countersReg:    make([]ustat.Counter, RegStatCounterItemMax),
		countersDereg:  make([]ustat.Counter, DeregStatCounterItemMax),
		countersSdmget: make([]ustat.Counter, SdmgetStatCounterItemMax),
		countersReach:  make([]ustat.Counter, ReachStatCounterItemMax),
		countersN1n2:   make([]ustat.Counter, N1n2StatCounterItemMax),
		countersSubs:   make([]ustat.Counter, SubsStatCounterItemMax),
		countersHttpif: make([]ustat.Counter, HttpifStatCounterItemMax),
	}

	// ERR
	for i := InvalidHttpMsg; i < ErrStatCounterItemMax; i++ {
		s.countersErr[i] = ustat.GetOrRegisterCounter(ErrStatCounterNames[i], s.registry)
		if s.countersErr[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, ErrStatCounterNames[i])
			return nil
		}
	}

	//Oper
	for i := ActivateStat; i < OperStatCounterItemMax; i++ {
		s.countersOper[i] = ustat.GetOrRegisterCounter(statCounterNames[i], s.registry)
		if s.countersOper[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, statCounterNames[i])
			return nil
		}
	}
	//////////////////
	for i := ActivateTotal; i < ActivateStatCounterItemMax; i++ {
		s.countersAct[i] = ustat.GetOrRegisterCounter(ActivateStatCounterNames[i], s.registry)
		if s.countersAct[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, ActivateStatCounterNames[i])
			return nil
		}
	}

	for i := DeactivateTotal; i < DeactivateStatCounterItemMax; i++ {
		s.countersDeact[i] = ustat.GetOrRegisterCounter(DeactivateStatCounterNames[i], s.registry)
		if s.countersDeact[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, DeactivateStatCounterNames[i])
			return nil
		}
	}

	for i := UplinkTotal; i < UplinkStatCounterItemMax; i++ {
		s.countersUplink[i] = ustat.GetOrRegisterCounter(UplinkStatCounterNames[i], s.registry)
		if s.countersUplink[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, UplinkStatCounterNames[i])
			return nil
		}
	}

	for i := RegTotal; i < RegStatCounterItemMax; i++ {
		s.countersReg[i] = ustat.GetOrRegisterCounter(RegStatCounterNames[i], s.registry)
		if s.countersReg[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, RegStatCounterNames[i])
			return nil
		}
	}

	for i := DeregTotal; i < DeregStatCounterItemMax; i++ {
		s.countersDereg[i] = ustat.GetOrRegisterCounter(DeregStatCounterNames[i], s.registry)
		if s.countersDereg[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, DeregStatCounterNames[i])
			return nil
		}
	}

	for i := SdmgetTotal; i < SdmgetStatCounterItemMax; i++ {
		s.countersSdmget[i] = ustat.GetOrRegisterCounter(SdmgetStatCounterNames[i], s.registry)
		if s.countersSdmget[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, SdmgetStatCounterNames[i])
			return nil
		}
	}

	for i := ReachTotal; i < ReachStatCounterItemMax; i++ {
		s.countersReach[i] = ustat.GetOrRegisterCounter(ReachStatCounterNames[i], s.registry)
		if s.countersReach[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, ReachStatCounterNames[i])
			return nil
		}
	}

	for i := N1n2Total; i < N1n2StatCounterItemMax; i++ {
		s.countersN1n2[i] = ustat.GetOrRegisterCounter(N1n2StatCounterNames[i], s.registry)
		if s.countersN1n2[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, N1n2StatCounterNames[i])
			return nil
		}
	}

	for i := SubsTotal; i < SubsStatCounterItemMax; i++ {
		s.countersSubs[i] = ustat.GetOrRegisterCounter(SubsStatCounterNames[i], s.registry)
		if s.countersSubs[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, SubsStatCounterNames[i])
			return nil
		}
	}

	for i := HttpifTotal; i < HttpifStatCounterItemMax; i++ {
		s.countersHttpif[i] = ustat.GetOrRegisterCounter(HttpifStatCounterNames[i], s.registry)
		if s.countersHttpif[i] == nil {
			loggers.ErrorLogger().Major("Failed to GetOrRegisterCounter(%v, %s)", i, HttpifStatCounterNames[i])
			return nil
		}
	}

	return s
}

// IncCounter 전달된 Item의 값을 증가 시킨다.
//func (s *Stats) IncCounter(item StatCounterItem) {
func (s *Stats) IncOperCounter(item StatCounterItem) {
	s.countersOper[item].Inc(1)
}

func (s *Stats) CheckOperCounter(item int) int64 {
	return s.countersOper[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddOperCounter(item StatCounterItem, count int64) {
	s.countersOper[item].Inc(count)
}

// IncCounter 전달된 Item의 값을 증가 시킨다.
//func (s *Stats) IncCounter(item StatCounterItem) {

////////// HTTP ERR
func (s *Stats) IncErrCounter(item ErrStatCounterItem) {
	s.countersErr[item].Inc(1)
}

func (s *Stats) CheckErrCounter(item int) int64 {
	return s.countersErr[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddErrCounter(item ErrStatCounterItem, count int64) {
	s.countersErr[item].Inc(count)
}

////////// ACT
func (s *Stats) IncActCounter(item ActivateStatCounterItem) {
	s.countersAct[item].Inc(1)
}

func (s *Stats) CheckActCounter(item int) int64 {
	return s.countersAct[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddActCounter(item ActivateStatCounterItem, count int64) {
	s.countersAct[item].Inc(count)
}

////////// DEACT
func (s *Stats) IncDeactCounter(item DeactivateStatCounterItem) {
	s.countersDeact[item].Inc(1)
}

func (s *Stats) CheckDeactCounter(item int) int64 {
	return s.countersDeact[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddDeactCounter(item DeactivateStatCounterItem, count int64) {
	s.countersDeact[item].Inc(count)
}

////////// UPLINK
func (s *Stats) IncUplinkCounter(item UplinkStatCounterItem) {
	s.countersUplink[item].Inc(1)
}

func (s *Stats) CheckUplinkCounter(item int) int64 {
	return s.countersUplink[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddUplinkCounter(item UplinkStatCounterItem, count int64) {
	s.countersUplink[item].Inc(count)
}

////////// Reg
func (s *Stats) IncRegCounter(item RegStatCounterItem) {
	s.countersReg[item].Inc(1)
}

func (s *Stats) CheckRegCounter(item int) int64 {
	return s.countersReg[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddRegCounter(item RegStatCounterItem, count int64) {
	s.countersReg[item].Inc(count)
}

////////// Dereg
func (s *Stats) IncDeregCounter(item DeregStatCounterItem) {
	s.countersDereg[item].Inc(1)
}

func (s *Stats) CheckDeregCounter(item int) int64 {
	return s.countersDereg[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddDeregCounter(item DeregStatCounterItem, count int64) {
	s.countersDereg[item].Inc(count)
}

////////// Sdmget
func (s *Stats) IncSdmgetCounter(item SdmgetStatCounterItem) {
	s.countersSdmget[item].Inc(1)
}

func (s *Stats) CheckSdmgetCounter(item int) int64 {
	return s.countersSdmget[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddSdmgetCounter(item SdmgetStatCounterItem, count int64) {
	s.countersSdmget[item].Inc(count)
}

////////// Reach
func (s *Stats) IncReachCounter(item ReachStatCounterItem) {
	s.countersReach[item].Inc(1)
}

func (s *Stats) CheckReachCounter(item int) int64 {
	return s.countersReach[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddReachCounter(item ReachStatCounterItem, count int64) {
	s.countersReach[item].Inc(count)
}

////////// N1n2
func (s *Stats) IncN1n2Counter(item N1n2StatCounterItem) {
	s.countersN1n2[item].Inc(1)
}

func (s *Stats) CheckN1n2Counter(item int) int64 {
	return s.countersN1n2[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddN1n2Counter(item N1n2StatCounterItem, count int64) {
	s.countersN1n2[item].Inc(count)
}

////////// Subs
func (s *Stats) IncSubsCounter(item SubsStatCounterItem) {
	s.countersSubs[item].Inc(1)
}

func (s *Stats) CheckSubsCounter(item int) int64 {
	return s.countersSubs[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddSubsCounter(item SubsStatCounterItem, count int64) {
	s.countersSubs[item].Inc(count)
}

////////////////////////////////////////////////////////////////////////////////
// StatCounterItem functions
////////////////////////////////////////////////////////////////////////////////

func (s StatCounterItem) String() string {
	return statCounterNames[s]
}

////////////////////////////////////////////////////////////////////////////////
// StatCounterItem functions
////////////////////////////////////////////////////////////////////////////////
func (s ErrStatCounterItem) String() string {
	return ErrStatCounterNames[s]
}

func (s ActivateStatCounterItem) String() string {
	return ActivateStatCounterNames[s]
}

func (s DeactivateStatCounterItem) String() string {
	return DeactivateStatCounterNames[s]
}

func (s UplinkStatCounterItem) String() string {
	return UplinkStatCounterNames[s]
}

func (s RegStatCounterItem) String() string {
	return RegStatCounterNames[s]
}

func (s DeregStatCounterItem) String() string {
	return DeregStatCounterNames[s]
}

func (s SdmgetStatCounterItem) String() string {
	return SdmgetStatCounterNames[s]
}

func (s ReachStatCounterItem) String() string {
	return ReachStatCounterNames[s]
}

func (s N1n2StatCounterItem) String() string {
	return N1n2StatCounterNames[s]
}

func (s SubsStatCounterItem) String() string {
	return SubsStatCounterNames[s]
}

////////////////////////////////////////////////////////////////////////////////
// Httpif functions
////////////////////////////////////////////////////////////////////////////////
func (s HttpifStatCounterItem) String() string {
	return HttpifStatCounterNames[s]
}

func (s *Stats) IncHttpifCounter(item HttpifStatCounterItem) {
	s.countersHttpif[item].Inc(1)
}

func (s *Stats) CheckHttpifCounter(item int) int64 {
	return s.countersHttpif[item].Count()
}

// AddCounter 전달된 Item의 값을 전달된 count 만큼 증가 시킨다.
func (s *Stats) AddHttpifCounter(item SubsStatCounterItem, count int64) {
	s.countersHttpif[item].Inc(count)
}
