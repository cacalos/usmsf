package controller

import (
	"bufio"
	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

type NFStat struct {
	stats *Stats
}

func NewStatWrite(cfg uconf.Config, stats *Stats) *NFStat {

	var conftime int
	var timer time.Duration

	var operFile *os.File
	var errFile *os.File
	var transFile *os.File
	var httpifFile *os.File
	var err error

	ErrFileTmpPath := fmt.Sprintf("%s/usmsf_svc_err_tmp", os.Getenv("SMS_STAT_DIR"))
	OperFileTmpPath := fmt.Sprintf("%s/usmsf_svc_oper_tmp", os.Getenv("SMS_STAT_DIR"))
	TransFileTmpPath := fmt.Sprintf("%s/usmsf_svc_trans_tmp", os.Getenv("SMS_STAT_DIR"))
	HttpifFileTmpPath := fmt.Sprintf("%s/usmsf_svc_httpif_tmp", os.Getenv("SMS_STAT_DIR"))

	ErrFilechgPath := fmt.Sprintf("%s/usmsf_svc_err", os.Getenv("SMS_STAT_DIR"))
	OperFilechgPath := fmt.Sprintf("%s/usmsf_svc_oper", os.Getenv("SMS_STAT_DIR"))
	TransFilechgPath := fmt.Sprintf("%s/usmsf_svc_trans", os.Getenv("SMS_STAT_DIR"))
	HttpifFilechgPath := fmt.Sprintf("%s/usmsf_svc_httpif", os.Getenv("SMS_STAT_DIR"))

	statTimer := cfg.GetConfig("statistic.stattimer")
	if statTimer != nil {
		conftime = statTimer.GetInt("timer", 10)
		timer = time.Duration(conftime)

	}

	if stats != nil {
		exec.SafeGo(func() {
			for {

				operFile, err = os.Create(OperFileTmpPath)
				if err != nil {
					os.MkdirAll(os.Getenv("SMS_STAT_DIR"), 0755)
					continue
				} else {
					errFile, err = os.Create(ErrFileTmpPath)
					if err != nil {
						panic(err)
					}
					transFile, err = os.Create(TransFileTmpPath)
					if err != nil {
						panic(err)
					}
					httpifFile, err = os.Create(HttpifFileTmpPath)
					if err != nil {
						panic(err)
					}

				}

				StatisticsErr(stats, ErrFileTmpPath, ErrFilechgPath, errFile)
				StatisticsOper(stats, OperFileTmpPath, OperFilechgPath, operFile)
				StatisticsTrans(stats, TransFileTmpPath, TransFilechgPath, transFile)
				StatisticsHttpif(stats, HttpifFileTmpPath, HttpifFilechgPath, httpifFile)
				time.Sleep(timer * time.Second)

			}
		})
	}

	s := &NFStat{
		stats: stats,
	}

	return s

}

func StatisticsErr(s *Stats, TmpPath string, ChgPath string, file *os.File) {

	var i int
	var cnt [ErrStatCounterItemMax]int64
	var strcnt [ErrStatCounterItemMax]string

	wr := csv.NewWriter(bufio.NewWriter(file))

	for i = InvalidHttpMsg; i < ErrStatCounterItemMax; i++ {
		cnt[i] = s.CheckErrCounter(i)

		strcnt[i] = strconv.FormatInt(cnt[i], 10)

	}
	wr.Write([]string{strcnt[InvalidHttpMsg], strcnt[InvalidToken], strcnt[InvalidSDMGETSendHttpMsg], strcnt[InvalidSDMSUBSSendHttpMsg], strcnt[InvalidUECMSendHttpMsg], strcnt[InvalidN1N2SendHttpMsg], strcnt[InvalidReachSendHttpMsg]})
	wr.Flush()

	err := os.Rename(TmpPath, ChgPath)
	if err != nil {
		panic(err)
	}
}

func StatisticsOper(s *Stats, TmpPath string, ChgPath string, file *os.File) {
	var i int
	//      var j int
	var cnt [9][ActivateStatCounterItemMax]int64
	var strcnt [9][ActivateStatCounterItemMax]string

	wr := csv.NewWriter(bufio.NewWriter(file))

	for i = ActivateStat; i < OperStatCounterItemMax; i++ {
		switch i {
		case ActivateStat:
			cnt[i][ActivateTotal] = s.CheckActCounter(ActivateTotal)
			strcnt[i][ActivateTotal] = strconv.FormatInt(cnt[i][ActivateTotal], 10)
			break
		case DeactivateStat:
			cnt[i][DeactivateTotal] = s.CheckDeactCounter(DeactivateTotal)
			strcnt[i][DeactivateTotal] = strconv.FormatInt(cnt[i][DeactivateTotal], 10)

			break
		case UplinkStat:
			cnt[i][UplinkTotal] = s.CheckUplinkCounter(UplinkTotal)
			strcnt[i][UplinkTotal] = strconv.FormatInt(cnt[i][UplinkTotal], 10)

			break
		case RegStat:
			cnt[i][RegTotal] = s.CheckRegCounter(RegTotal)
			strcnt[i][RegTotal] = strconv.FormatInt(cnt[i][RegTotal], 10)

			break
		case DeregStat:
			cnt[i][DeregTotal] = s.CheckDeregCounter(DeregTotal)
			strcnt[i][DeregTotal] = strconv.FormatInt(cnt[i][DeregTotal], 10)

			break
		case SdmgetStat:
			cnt[i][SdmgetTotal] = s.CheckSdmgetCounter(SdmgetTotal)
			strcnt[i][SdmgetTotal] = strconv.FormatInt(cnt[i][SdmgetTotal], 10)

			break
		case ReachStat:
			cnt[i][ReachTotal] = s.CheckReachCounter(ReachTotal)
			strcnt[i][ReachTotal] = strconv.FormatInt(cnt[i][ReachTotal], 10)
			break
		case N1n2Stat:
			cnt[i][N1n2Total] = s.CheckN1n2Counter(N1n2Total)
			strcnt[i][N1n2Total] = strconv.FormatInt(cnt[i][N1n2Total], 10)

			break
		case SubsStat:
			cnt[i][SubsTotal] = s.CheckSubsCounter(SubsTotal)
			strcnt[i][SubsTotal] = strconv.FormatInt(cnt[i][SubsTotal], 10)
			break
		default:
			break
		}
	}
	wr.Write([]string{strcnt[0][0], strcnt[1][0], strcnt[2][0], strcnt[3][0], strcnt[4][0], strcnt[5][0], strcnt[6][0], strcnt[7][0], strcnt[8][0]})
	wr.Flush()

	err := os.Rename(TmpPath, ChgPath)
	if err != nil {
		panic(err)
	}
}

func StatisticsTrans(s *Stats, TmpPath string, ChgPath string, file *os.File) {

	var i int
	var j int
	var cnt [9][ActivateStatCounterItemMax]int64
	var strcnt [9][ActivateStatCounterItemMax]string

	wr := csv.NewWriter(bufio.NewWriter(file))

	for i = 0; i < 9; i++ {
		switch i {
		case ActivateStat:
			for j = 0; j < ActivateStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckActCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"act", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case DeactivateStat:
			for j = 0; j < DeactivateStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckDeactCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"deact", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case UplinkStat:
			for j = 0; j < UplinkStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckUplinkCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"uplink", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case RegStat:
			for j = 0; j < RegStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckRegCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"reg", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case DeregStat:
			for j = 0; j < DeregStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckDeregCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"dereg", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case SdmgetStat:
			for j = 0; j < SdmgetStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckSdmgetCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"sdm_get", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case ReachStat:
			for j = 0; j < ReachStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckReachCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"reach", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case N1n2Stat:
			for j = 0; j < N1n2StatCounterItemMax; j++ {
				cnt[i][j] = s.CheckN1n2Counter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"n1n2", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		case SubsStat:
			for j = 0; j < SubsStatCounterItemMax; j++ {
				cnt[i][j] = s.CheckSubsCounter(j)
				strcnt[i][j] = strconv.FormatInt(cnt[i][j], 10)
			}

			wr.Write([]string{"subs", strcnt[i][0], strcnt[i][1], strcnt[i][2], strcnt[i][3], strcnt[i][4], strcnt[i][5], strcnt[i][6], strcnt[i][7], strcnt[i][8], strcnt[i][9], strcnt[i][10], strcnt[i][11]})
			wr.Flush()
			break
		default:
			wr.Flush()
			break
		}
	}
	err := os.Rename(TmpPath, ChgPath)
	if err != nil {
		panic(err)
	}
}

func StatisticsHttpif(s *Stats, TmpPath string, ChgPath string, file *os.File) {

	var i int
	var cnt [HttpifStatCounterItemMax]int64
	var strcnt [HttpifStatCounterItemMax]string

	wr := csv.NewWriter(bufio.NewWriter(file))

	for i = HttpifTotal; i < HttpifStatCounterItemMax; i++ {
		cnt[i] = s.CheckHttpifCounter(i)

		strcnt[i] = strconv.FormatInt(cnt[i], 10)

	}
	wr.Write([]string{strcnt[HttpifTotal], strcnt[MoResp]})
	wr.Flush()

	err := os.Rename(TmpPath, ChgPath)
	if err != nil {
		panic(err)
	}
}

func StatDivideOperation(oper string) int {
	switch oper {
	case "PUT":
		return ActivateStat
	case "DELETE":
		return DeactivateStat
	case "POST":
		return UplinkStat
	}

	return -1

}

func (s *MapServer) StatHttpif() {
	s.stats.IncHttpifCounter(HttpifTotal)
	s.stats.IncHttpifCounter(MoResp)
}

func (s *MapServer) StatHttpRespCode(respcode int, flag int) {
	switch flag {

	case ReachStat:
		switch respcode {
		case 200:
			s.stats.IncReachCounter(Reach200)
			break
		case 201:
			s.stats.IncReachCounter(Reach201)
			break
		case 202:
			s.stats.IncReachCounter(Reach202)
			break
		case 204:
			s.stats.IncReachCounter(Reach204)
			break
		case 400:
			s.stats.IncReachCounter(Reach400)
			break
		case 403:
			s.stats.IncReachCounter(Reach403)
			break
		case 404:
			s.stats.IncReachCounter(Reach404)
			break
		case 501:
			s.stats.IncReachCounter(Reach501)
			break
		case 503:
			s.stats.IncReachCounter(Reach503)
			break
		case 504:
			s.stats.IncReachCounter(Reach504)
			break
		default:
			s.stats.IncReachCounter(ReachEtcResp)
			break
		}
		break
	case N1n2Stat:
		switch respcode {
		case 200:
			s.stats.IncN1n2Counter(N1n2200)
			break
		case 201:
			s.stats.IncN1n2Counter(N1n2201)
			break
		case 202:
			s.stats.IncN1n2Counter(N1n2202)
			break
		case 204:
			s.stats.IncN1n2Counter(N1n2204)
			break
		case 400:
			s.stats.IncN1n2Counter(N1n2400)
			break
		case 403:
			s.stats.IncN1n2Counter(N1n2403)
			break
		case 404:
			s.stats.IncN1n2Counter(N1n2404)
			break
		case 501:
			s.stats.IncN1n2Counter(N1n2501)
			break
		case 503:
			s.stats.IncN1n2Counter(N1n2503)
			break
		case 504:
			s.stats.IncN1n2Counter(N1n2504)
			break
		default:
			s.stats.IncN1n2Counter(N1n2EtcResp)
			break
		}
		break

	}

}

func (s *NFServer) StatHttpRespCode(respcode int, flag int) {

	switch flag {
	case ActivateStat:
		switch respcode {
		case 200:
			s.stats.IncActCounter(Activate200)
			break
		case 201:
			s.stats.IncActCounter(Activate201)
			break
		case 202:
			s.stats.IncActCounter(Activate202)
			break
		case 204:
			s.stats.IncActCounter(Activate204)
			break
		case 400:
			s.stats.IncActCounter(Activate400)
			break
		case 403:
			s.stats.IncActCounter(Activate403)
			break
		case 404:
			s.stats.IncActCounter(Activate404)
			break
		case 501:
			s.stats.IncActCounter(Activate501)
			break
		case 503:
			s.stats.IncActCounter(Activate503)
			break
		case 504:
			s.stats.IncActCounter(Activate504)
			break
		default:
			s.stats.IncActCounter(ActivateEtcResp)
			break
		}
		break
	case DeactivateStat:
		switch respcode {
		case 200:
			s.stats.IncDeactCounter(Deactivate200)
			break
		case 201:
			s.stats.IncDeactCounter(Deactivate201)
			break
		case 202:
			s.stats.IncDeactCounter(Deactivate202)
			break
		case 204:
			s.stats.IncDeactCounter(Deactivate204)
			break
		case 400:
			s.stats.IncDeactCounter(Deactivate400)
			break
		case 403:
			s.stats.IncDeactCounter(Deactivate403)
			break
		case 404:
			s.stats.IncDeactCounter(Deactivate404)
			break
		case 501:
			s.stats.IncDeactCounter(Deactivate501)
			break
		case 503:
			s.stats.IncDeactCounter(Deactivate503)
			break
		case 504:
			s.stats.IncDeactCounter(Deactivate504)
			break
		default:
			s.stats.IncDeactCounter(DeactivateEtcResp)
			break
		}
		break
	case UplinkStat:
		switch respcode {
		case 200:
			s.stats.IncUplinkCounter(Uplink200)
			break
		case 201:
			s.stats.IncUplinkCounter(Uplink201)
			break
		case 202:
			s.stats.IncUplinkCounter(Uplink202)
			break
		case 204:
			s.stats.IncUplinkCounter(Uplink204)
			break
		case 400:
			s.stats.IncUplinkCounter(Uplink400)
			break
		case 403:
			s.stats.IncUplinkCounter(Uplink403)
			break
		case 404:
			s.stats.IncUplinkCounter(Uplink404)
			break
		case 501:
			s.stats.IncUplinkCounter(Uplink501)
			break
		case 503:
			s.stats.IncUplinkCounter(Uplink503)
			break
		case 504:
			s.stats.IncUplinkCounter(Uplink504)
			break
		default:
			s.stats.IncUplinkCounter(UplinkEtcResp)
			break
		}
		break
	case RegStat:
		switch respcode {
		case 200:
			s.stats.IncRegCounter(Reg200)
			break
		case 201:
			s.stats.IncRegCounter(Reg201)
			break
		case 202:
			s.stats.IncRegCounter(Reg202)
			break
		case 204:
			s.stats.IncRegCounter(Reg204)
			break
		case 400:
			s.stats.IncRegCounter(Reg400)
			break
		case 403:
			s.stats.IncRegCounter(Reg403)
			break
		case 404:
			s.stats.IncRegCounter(Reg404)
			break
		case 501:
			s.stats.IncRegCounter(Reg501)
			break
		case 503:
			s.stats.IncRegCounter(Reg503)
			break
		case 504:
			s.stats.IncRegCounter(Reg504)
			break
		default:
			s.stats.IncRegCounter(RegEtcResp)
			break
		}
		break
	case DeregStat:
		switch respcode {
		case 200:
			s.stats.IncDeregCounter(Dereg200)
			break
		case 201:
			s.stats.IncDeregCounter(Dereg201)
			break
		case 202:
			s.stats.IncDeregCounter(Dereg202)
			break
		case 204:
			s.stats.IncDeregCounter(Dereg204)
			break
		case 400:
			s.stats.IncDeregCounter(Dereg400)
			break
		case 403:
			s.stats.IncDeregCounter(Dereg403)
			break
		case 404:
			s.stats.IncDeregCounter(Dereg404)
			break
		case 501:
			s.stats.IncDeregCounter(Dereg501)
			break
		case 503:
			s.stats.IncDeregCounter(Dereg503)
			break
		case 504:
			s.stats.IncDeregCounter(Dereg504)
			break
		default:
			s.stats.IncDeregCounter(DeregEtcResp)
			break
		}
		break
	case SdmgetStat:
		switch respcode {
		case 200:
			s.stats.IncSdmgetCounter(Sdmget200)
			break
		case 201:
			s.stats.IncSdmgetCounter(Sdmget201)
			break
		case 202:
			s.stats.IncSdmgetCounter(Sdmget202)
			break
		case 204:
			s.stats.IncSdmgetCounter(Sdmget204)
			break
		case 400:
			s.stats.IncSdmgetCounter(Sdmget400)
			break
		case 403:
			s.stats.IncSdmgetCounter(Sdmget403)
			break
		case 404:
			s.stats.IncSdmgetCounter(Sdmget404)
			break
		case 501:
			s.stats.IncSdmgetCounter(Sdmget501)
			break
		case 503:
			s.stats.IncSdmgetCounter(Sdmget503)
			break
		case 504:
			s.stats.IncSdmgetCounter(Sdmget504)
			break
		default:
			s.stats.IncSdmgetCounter(SdmgetEtcResp)
			break
		}
		break
	case ReachStat:
		switch respcode {
		case 200:
			s.stats.IncReachCounter(Reach200)
			break
		case 201:
			s.stats.IncReachCounter(Reach201)
			break
		case 202:
			s.stats.IncReachCounter(Reach202)
			break
		case 204:
			s.stats.IncReachCounter(Reach204)
			break
		case 400:
			s.stats.IncReachCounter(Reach400)
			break
		case 403:
			s.stats.IncReachCounter(Reach403)
			break
		case 404:
			s.stats.IncReachCounter(Reach404)
			break
		case 501:
			s.stats.IncReachCounter(Reach501)
			break
		case 503:
			s.stats.IncReachCounter(Reach503)
			break
		case 504:
			s.stats.IncReachCounter(Reach504)
			break
		default:
			s.stats.IncReachCounter(ReachEtcResp)
			break
		}
		break
	case N1n2Stat:
		switch respcode {
		case 200:
			s.stats.IncN1n2Counter(N1n2200)
			break
		case 201:
			s.stats.IncN1n2Counter(N1n2201)
			break
		case 202:
			s.stats.IncN1n2Counter(N1n2202)
			break
		case 204:
			s.stats.IncN1n2Counter(N1n2204)
			break
		case 400:
			s.stats.IncN1n2Counter(N1n2400)
			break
		case 403:
			s.stats.IncN1n2Counter(N1n2403)
			break
		case 404:
			s.stats.IncN1n2Counter(N1n2404)
			break
		case 501:
			s.stats.IncN1n2Counter(N1n2501)
			break
		case 503:
			s.stats.IncN1n2Counter(N1n2503)
			break
		case 504:
			s.stats.IncN1n2Counter(N1n2504)
			break
		default:
			s.stats.IncN1n2Counter(N1n2EtcResp)
			break
		}
		break
	case SubsStat:
		switch respcode {
		case 200:
			s.stats.IncSubsCounter(Subs200)
			break
		case 201:
			s.stats.IncSubsCounter(Subs201)
			break
		case 202:
			s.stats.IncSubsCounter(Subs202)
			break
		case 204:
			s.stats.IncSubsCounter(Subs204)
			break
		case 400:
			s.stats.IncSubsCounter(Subs400)
			break
		case 403:
			s.stats.IncSubsCounter(Subs403)
			break
		case 404:
			s.stats.IncSubsCounter(Subs404)
			break
		case 501:
			s.stats.IncSubsCounter(Subs501)
			break
		case 503:
			s.stats.IncSubsCounter(Subs503)
			break
		case 504:
			s.stats.IncSubsCounter(Subs504)
			break
		default:
			s.stats.IncSubsCounter(SubsEtcResp)
			break
		}
		break
	default:
		break
	}
}
