package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"camel.uangel.com/ua5g/scpcli.git"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/utypes"
	"golang.org/x/xerrors"

	"camel.uangel.com/ua5g/usmsf.git/dao"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
)

func (s *NFServer) Uecm_reig(
	supi string,
	amfSupi string,
	smsContext *msg5g.UeSmsContextData,
) error {
	/* send UECM Req to UDM for test */
	var uecmURL string
	//var buf []byte

	rval, buf := s.redisDao.GetSubInfoBySUPI(amfSupi, "accessType")
	tBuf := string(buf[:])
	accessType := strings.Trim(tBuf, "\"")
	if rval != 1 {
		accessType = "3GPP_ACCESS"
	}

	// Make UECM_Regi Request URL
	if accessType == "3GPP_ACCESS" {
		uecmURL = fmt.Sprintf("/%s/registrations/smsf-3gpp-access", supi)
	} else {
		uecmURL = fmt.Sprintf("/%s/registrations/smsf-non-3gpp-access", supi)
	}

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	// Make UECM Body
	var uecm uclient.MakeMarshalIndent

	uecm = &msg5g.UECM{s.nfId, s.isdn, s.mnc, s.mcc, s.name, s.realm}

	RegContext := context.Background()
	if s.traceInfo.OnOff == true {
		RegContext = uclient.ContextWithTraceLabel(RegContext, "supi", supi)
	}

	loggers.InfoLogger().Comment("UECM_REGI Req. -> %s", uecmURL)
	res, err := s.udmCli.ServiceRequest(RegContext,
		"nudm-uecm",
		utypes.Map{
			"supi":             smsContext.Supi,
			"udmGroupId":       smsContext.UdmGroupId,
			"gpsi":             smsContext.Gpsi,
			"routingIndicator": smsContext.RoutingIndicator,
		},
		"PUT",
		uecmURL,
		hdr,
		uecm,
		time.Second*10,
	)

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Nudm-UECM(SMSF Registration) PUT err : %s, USER:%s", err, supi)
		callLogger.Info("SEND FAIL REGI TO UDM, ERR:%s, USER=%s", err, supi)
		s.stats.IncErrCounter(InvalidUECMSendHttpMsg)
		return err

	} else if err != nil {
		loggers.ErrorLogger().Major("Send Fail Nudm-UECM(SMSF Registration) PUT err : %s, USER:%s", err, supi)
		callLogger.Info("SEND FAIL REGI TO UDM, ERR:%s, USER=%s", err, supi)
		s.StatHttpRespCode(http.StatusForbidden, ActivateStat)
		return err
	}

	loggers.InfoLogger().Comment("Send Succ Nudm-EUCM(SMSF Registration) PUT, USER:%s", supi)
	callLogger.Info("SEND SUCC REG) TO UDM, USER=%s", supi)

	if res.StatusCode() > 300 {
		loggers.ErrorLogger().Minor("Recv Nudm-EUCM(SMSF Registration) Resp(NACK) Result Code : %d(%s), USER:%s", res.StatusCode(), res.ResponseString(), supi)
		callLogger.Info("RECV REGI RESP FROM UDM, RESP:%d, USER=%s", res.StatusCode(), supi)

		s.StatHttpRespCode(res.StatusCode(), RegStat)
		s.StatHttpRespCode(http.StatusForbidden, ActivateStat)

		err := errors.New("Service Not Allowed")
		return err
	}

	s.stats.IncRegCounter(RegTotal)
	loggers.InfoLogger().Comment("Recv Nudm-EUCM(SMSF Registration) Resp(ACK), Result Code : %d, USER:%s", res.StatusCode(), supi)
	callLogger.Info("RECV REGI RESP FROM UDM, RESP:%d, USER=%s", res.StatusCode(), supi)

	return nil
}

func (s *NFServer) Sdm_Get(supi string,
	smsContext *msg5g.UeSmsContextData,
) (respCode int,
	err error,
	sdmRespData []byte,
	sdmResp *msg5g.SmsManagementSubscriptionData) {

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	/* send SDM Get to UDM */
	sdmURL := fmt.Sprintf("/%s/sms-mng-data", supi)

	SdmContext := context.Background()
	if s.traceInfo.OnOff == true {
		SdmContext = uclient.ContextWithTraceLabel(SdmContext, "supi", supi)
	}
	res, err := s.udmCli.ServiceRequest(SdmContext,
		"nudm-sdm",
		utypes.Map{
			"supi":             smsContext.Supi,
			"udmGroupId":       smsContext.UdmGroupId,
			"gpsi":             smsContext.Gpsi,
			"routingIndicator": smsContext.RoutingIndicator,
		},
		"GET",
		sdmURL,
		hdr,
		"",
		time.Second*10,
	)

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Nudm-SDM(SMS-MNG-DATA) GET err : %s, USER:%s", err, supi)
		callLogger.Info("SEND FAIL SDM-GET TO UDM, ERR:%s, USER=%s", err, supi)
		s.stats.IncErrCounter(InvalidSDMGETSendHttpMsg)

		return respCode, err, sdmRespData, sdmResp
	}

	loggers.InfoLogger().Comment("Send Succ Nudm-SDM(SMS-MNG-DATA) GET, USER:%s", supi)
	callLogger.Info("SEND SUCC SDM-GET TO UDM, USER=%s", supi)

	if res.StatusCode() > 300 {
		loggers.ErrorLogger().Minor("Recv Nudm-SDM(SMS-MNG-DATA) Resp(NACK) Result Code : %d(%s), USER:%s", res.StatusCode(), res.ResponseString(), supi)
		callLogger.Info("RECV SDM-GET RESP FROM UDM, RESP:%d, USER=%s", res.StatusCode(), supi)

		if res.StatusCode() == http.StatusNotFound {
			return res.StatusCode(), err, sdmRespData, sdmResp
		}
		s.StatHttpRespCode(http.StatusForbidden, ActivateStat)
		err := errors.New("Service Not Allowed")
		return respCode, err, sdmRespData, sdmResp
	}

	loggers.InfoLogger().Comment("Recv Nudm-SDM(SMS-MNG-DATA) Resp(ACK) Result Code : %d , USER:%s", res.StatusCode(), supi)
	callLogger.Info("RECV SDM-GET RESP FROM UDM, RESP:%d, USER=%s", res.StatusCode(), supi)
	sdmRespData = res.Response().([]byte)

	s.stats.IncSdmgetCounter(SdmgetTotal)
	s.StatHttpRespCode(res.StatusCode(), SdmgetStat)

	loggers.InfoLogger().Comment("SDM_GET resp, USER:%s, BODY : %s", supi, string(sdmRespData))

	sdmResp = new(msg5g.SmsManagementSubscriptionData)

	err = json.Unmarshal(sdmRespData, &sdmResp)
	if err != nil {
		s.StatHttpRespCode(http.StatusForbidden, ActivateStat)
		return respCode, err, sdmRespData, sdmResp
	}

	return respCode, err, sdmRespData, sdmResp
}

func (s *NFServer) Sdm_Subscription(
	supi string,
	amfSupi string,
	smsContext *msg5g.UeSmsContextData,
	sdmRespData []byte,
	sdmResp *msg5g.SmsManagementSubscriptionData,
	body []byte,
) error {
	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	// Make SDM Subscription Body
	var sdmsubs uclient.MakeMarshalIndent
	sdmsubs = &msg5g.SDM{s.nfId, s.notifyUrlAddr, supi}
	//subReqBody, err := sdm.Make()

	SubsContext := context.Background()
	if s.traceInfo.OnOff == true {
		SubsContext = uclient.ContextWithTraceLabel(SubsContext, "supi", supi)
	}

	// Make SDM_Subscription Request URL
	scrURL := fmt.Sprintf("/%s/sdm-subscriptions", supi)

	// Send Request UDM_UECM
	res, err := s.udmCli.ServiceRequest(SubsContext,
		"nudm-sdm",
		utypes.Map{
			"supi":             smsContext.Supi,
			"udmGroupId":       smsContext.UdmGroupId,
			"gpsi":             smsContext.Gpsi,
			"routingIndicator": smsContext.RoutingIndicator,
		},
		"POST",
		scrURL,
		hdr,
		sdmsubs,
		time.Second*10,
	)

	if err != nil && xerrors.Is(err, scpcli.ServiceUnavailable) {
		loggers.ErrorLogger().Major("Send Fail Nudm-SDM(SDM-Subscriptions) POST err : %s, USER:%s", err, supi)
		callLogger.Info("SEND FAIL SDM-SUBSCRIPTION TO UDM, ERR:%s, USER=%s", err, supi)
		s.stats.IncErrCounter(InvalidSDMSUBSSendHttpMsg)
		return err
	} else {
		loggers.InfoLogger().Comment("Send Succ Nudm-SDM(SDM-Subscriptions) POST, USER:%s", supi)
		callLogger.Info("SEND SUCC SDM-SUBSCRIPTION TO UDM, USER=%s", supi)
	}

	if res.StatusCode() >= 300 {
		loggers.ErrorLogger().Minor("Recv Nudm-SDM(SDM-Subscriptions) Resp(NACK) Result Code : %d(%s), USER:%s", res.StatusCode(), res.ResponseString(), supi)
	} else {
		loggers.InfoLogger().Comment("Recv Nudm-SDM(SDM-Subscriptions) Resp(ACK) Result Code : %d , USER:%s", res.StatusCode(), supi)

		subRespData := res.Response().([]byte)
		subResp := new(msg5g.SdmSubscribction)

		err = json.Unmarshal(subRespData, &subResp)

		//sdm = msg5g.SDM{s.nfId, s.notifyUrlAddr, supi}
		subInfo := msg5g.SubInfo{
			sdmResp.SupportFeatures,
			sdmResp.MtSmsSubscribed,
			sdmResp.MtSmsBarringAll,
			sdmResp.MtSmsBarringRoaming,
			sdmResp.MoSmsSubscribed,
			sdmResp.MoSmsBarringAll,
			sdmResp.MoSmsBarringRoaming,
			subResp.SubscriptionID,
		}

		subData, _ := json.Marshal(subInfo)

		/* insert subscriber in redis */
		sdmSupi := "sdm-" + supi
		rval := s.redisDao.InsSub(sdmSupi, subData)
		if rval == -1 {
			loggers.ErrorLogger().Major("redis_delete fail(), SDM user:%s", supi)
			rval = s.redisDao.DelSub(amfSupi)
			if rval == -1 {
				loggers.ErrorLogger().Major("redis_delete fail(), AMF user:%s", supi)
			} else {
				s.mysqlDao.Delete(amfSupi)
			}

			s.StatHttpRespCode(http.StatusForbidden, ActivateStat)

			return errors.New("Service Not Allowed")
		}

		mbody := &dao.MariaInfo{IMSI: sdmSupi, DATA: sdmRespData}
		s.mysqlDao.Create(mbody)

		s.stats.IncSubsCounter(SubsTotal)
		s.StatHttpRespCode(res.StatusCode(), SubsStat)
	}

	return nil
}
