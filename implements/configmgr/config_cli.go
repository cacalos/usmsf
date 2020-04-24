/**
 * @brief All Action of Configuration
 * @author parkjh
 * @file config.go
 * @data 2019-06-13
 * @version 0.1
 */
package configmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func (c *ConfigClient) GetDecisionConfig() error {
	loggers.InfoLogger().Comment("Initial Get Decision(sigtran or diameter) Config Start..")

	dicisionMsg := []DecisionConfigRespData{}

	var baseURL string
	baseURL = fmt.Sprintf("/app/v1/configurations")
	rselecGetURL, err := url.Parse(baseURL)
	if err != nil {
		loggers.ErrorLogger().Major("Create Uri Path")
		return err
	}

	// Prepare Query Parameters
	params := url.Values{}
	params.Add("cn", decision_config_name)
	params.Add("sn", service_name)

	// Add Query Parameters to the URL
	rselecGetURL.RawQuery = params.Encode()

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	loggers.InfoLogger().Comment("Send To ConfigMgr for DecisionConfiguration")

	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(), rselecGetURL.String(), "GET", hdr, nil, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("CONFIGURATIONS GetResp Response err() : %s ", err.Error())
			return err
		}

		loggers.InfoLogger().Comment("ConfigMgr-Recv ResponseData(Decision Info) : %s", string(GetResp.Response.([]byte)))

		err = json.Unmarshal(GetResp.Response.([]byte), &dicisionMsg)
		if err != nil {
			loggers.ErrorLogger().Major("JSON unmarshalling Error(Decision Config) : %s", err.Error())
			return err
		}

		for _, value := range dicisionMsg {
			loggers.InfoLogger().Comment("Decision StoragePush Data")
			err = c.ConfigIdGET(value.ConfId, decision)
			if err != nil {
				loggers.ErrorLogger().Major("Decision Error : %v", err)
				return err
			}
			SetConfIdStorage(value.ConfId)
		}

		loggers.InfoLogger().Comment("Select Map or Dia Configuration Count[%d] Result Code : %d", len(dicisionMsg), GetResp.StatusCode)
	} else {
		loggers.InfoLogger().Comment("Send URI : %s%s", c.cli.RootPath, rselecGetURL.String())
		GetResp, respData, err := c.cli.Call("GET", rselecGetURL.String(), hdr, nil)

		if err != nil {
			loggers.ErrorLogger().Major("http Request Fail")
			return err
		}

		loggers.InfoLogger().Comment("RecvDataJson(Dicision) : RespCode %d - %s",
			GetResp.StatusCode, string(respData))

		if GetResp.StatusCode > 300 {
			return errors.New(fmt.Sprintf("Response Code Error(%d)",
				GetResp.StatusCode))
		}

		err = json.Unmarshal(respData, &dicisionMsg)
		if err != nil {
			return err
		}

		for _, value := range dicisionMsg {
			loggers.InfoLogger().Comment("Decision StoragePush Data")
			err = c.ConfigIdGET(value.ConfId, decision)
			if err != nil {
				//		loggers.ErrorLogger().Major("Decision Error : %v", err)
				return err
			}
			SetConfIdStorage(value.ConfId)
		}

		loggers.InfoLogger().Comment("Select Map or Dia Configuration Count[%d] Result Code : %d",
			len(dicisionMsg), GetResp.StatusCode)

	}

	return nil
}

func (c *ConfigClient) GetCommonConfig() error {
	loggers.InfoLogger().Comment("Initial Get Common Config Start..")

	var commonbaseURL string
	commonMsg := []CommonConfigRespData{}

	commonbaseURL = fmt.Sprintf("/app/v1/configurations")
	commonGetURL, err := url.Parse(commonbaseURL)
	if err != nil {
		loggers.ErrorLogger().Major("Malformed URL : %s", err.Error())
		return err
	}

	// Prepare Query Parameters
	params := url.Values{}
	params.Add("cn", common_config_name)
	params.Add("sn", service_name)

	// Add Query Parameters to the URL
	commonGetURL.RawQuery = params.Encode()

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	loggers.InfoLogger().Comment("Send To ConfigMgr for CommonConfiguration")
	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(), commonGetURL.String(), "GET", hdr, nil, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("CONFIGURATIONS GetResp Response err() : %s ", err.Error())
			return err
		}

		loggers.InfoLogger().Comment("ConfigMgr-Recv ResponseData : %s", string(GetResp.Response.([]byte)))

		err = json.Unmarshal(GetResp.Response.([]byte), &commonMsg)
		if err != nil {
			loggers.ErrorLogger().Major("JSON unmarshalling Error(CommonConfig) : %s", err.Error())
			return err
		}

		// Delete reason.. Common Configuration Config File Only One.
		loggers.InfoLogger().Comment("CommonStoragePush Data")

		c.ConfigIdGET(commonMsg[0].ConfId, smsf)
		SetConfIdStorage(commonMsg[0].ConfId)

		loggers.InfoLogger().Comment("Common Configuration Result Code : %d", GetResp.StatusCode)
	} else {
		GetResp, respData, err := c.cli.Call("GET", commonGetURL.String(), hdr, nil)

		if err != nil {
			loggers.ErrorLogger().Major("GET Common Config ERROR")
			return err
		}

		loggers.InfoLogger().Comment("RecvDataJson(SMSF) : RespCode %d - %s", GetResp.StatusCode, string(respData))
		err = json.Unmarshal(respData, &commonMsg)
		if err != nil {
			loggers.ErrorLogger().Major("JSON unmarshalling Error(CommonConfig) : %s", err.Error())
			return err
		}

		// Delete reason.. Common Configuration Config File Only One.
		loggers.InfoLogger().Comment("CommonStoragePush Data")

		c.ConfigIdGET(commonMsg[0].ConfId, smsf)
		SetConfIdStorage(commonMsg[0].ConfId)

		loggers.InfoLogger().Comment("Common Configuration Result Code : %d", GetResp.StatusCode)

	}

	return nil

}

func (c *ConfigClient) GetSmscConfig() error {
	loggers.InfoLogger().Comment("Initial Get SMSC Config Start..")
	var smscbaseURL string
	smscMsg := []SmscConfigRespData{}

	smscbaseURL = fmt.Sprintf("/app/v1/configurations")
	smscGetURL, err := url.Parse(smscbaseURL)
	if err != nil {
		loggers.ErrorLogger().Major("Malformed URL : %s", err.Error())
		return err
	}

	// Prepare Query Parameters
	params := url.Values{}
	params.Add("cn", smscinfo_config_name)
	params.Add("sn", service_name)

	// Add Query Parameters to the URL
	smscGetURL.RawQuery = params.Encode()

	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")

	loggers.InfoLogger().Comment("Send To ConfigMgr for SmscConfigRespData")

	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(), smscGetURL.String(), "GET", hdr, nil, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("CONFIGURATIONS GetResp Response err() : %s ", err.Error())
			return err
		}

		loggers.InfoLogger().Comment("ConfigMgr-Recv ResponseData(SMSC Info) : %s", string(GetResp.Response.([]byte)))

		err = json.Unmarshal(GetResp.Response.([]byte), &smscMsg)
		if err != nil {
			loggers.ErrorLogger().Major("JSON unmarshalling Error(SMSC Config) : %s", err.Error())
			return err
		}

		for _, value := range smscMsg {
			loggers.InfoLogger().Comment("SMSCStoragePush Data")
			c.ConfigIdGET(value.ConfId, smsc)
			SetConfIdStorage(value.ConfId)
		}

		loggers.InfoLogger().Comment("SMSC Table Info Configuration Count[%d] Result Code : %d", len(smscMsg), GetResp.StatusCode)
	} else {

		GetResp, respData, err := c.cli.Call("GET", smscGetURL.String(), hdr, nil)

		if err != nil {
			return err
		}

		loggers.InfoLogger().Comment("RecvDataJson(SMSC) : RespCode %d - %s", GetResp.StatusCode, string(respData))
		err = json.Unmarshal(respData, &smscMsg)
		if err != nil {
			loggers.ErrorLogger().Major("JSON unmarshalling Error(SMSC Config) : %s", err.Error())
			return err
		}

		for _, value := range smscMsg {
			loggers.InfoLogger().Comment("SMSCStoragePush Data")
			c.ConfigIdGET(value.ConfId, smsc)
			SetConfIdStorage(value.ConfId)
		}

		loggers.InfoLogger().Comment("SMSC Table Info Configuration Count[%d] Result Code : %d", len(smscMsg), GetResp.StatusCode)

	}

	return nil

}

func (c *ConfigClient) ConfigIdGET(confId string, configType int) error {
	loggers.InfoLogger().Comment("Call Function ConfigId  GET Start..")

	var baseURL string
	baseURL = fmt.Sprintf("/app/v1/configurations/%s", confId)

	GetURL, err := url.Parse(baseURL)
	if err != nil {
		loggers.ErrorLogger().Major("Malformed URL : %s", err.Error())
		return err
	}

	// Prepare Query Parameters
	params := url.Values{}

	// Add Query Parameters to the URL
	GetURL.RawQuery = params.Encode()

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(), GetURL.String(), "GET", hdr, nil, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("CONFIGURATIONS GetResp Response err() : %s ", err.Error())
			return err
		}

		switch configType {
		case decision:
			Msg := &DecisionConfigRespData{}

			loggers.InfoLogger().Comment("Decision Msg : %s", string(GetResp.Response.([]byte)))
			err = json.Unmarshal(GetResp.Response.([]byte), Msg)
			if err != nil {
				return err
			}

			DecisionStoragePush(Msg.ConfId, Msg.Configuration)
			loggers.InfoLogger().Comment("Save Decision StoragePush Data")
			break

		case smsf:
			Msg := &CommonConfigRespData{}

			loggers.InfoLogger().Comment("Smsf Msg : %s", string(GetResp.Response.([]byte)))
			err = json.Unmarshal(GetResp.Response.([]byte), &Msg)
			if err != nil {
				return err
			}
			loggers.InfoLogger().Comment("Save CommonStoragePush Data")
			CommonStoragePush(Msg.ConfId, Msg.Configuration)
			break

		case smsc:
			Msg := &SmscConfigRespData{}

			loggers.InfoLogger().Comment("Smsc Msg : %s", string(GetResp.Response.([]byte)))
			err = json.Unmarshal(GetResp.Response.([]byte), &Msg)
			if err != nil {
				return err
			}

			loggers.InfoLogger().Comment("Save SMSCStoragePush Data")
			SmscStoragePush(Msg.ConfId, Msg.Configuration)
			break

		default:
			loggers.ErrorLogger().Major("Invalid Configuration ID : %s", confId)
			return errors.New(fmt.Sprintf("Invalid Configuration ID : %s", confId))

		}
		loggers.InfoLogger().Comment("Configuration Count Result Code : %d", GetResp.StatusCode)
	} else {

		GetResp, respData, err := c.cli.Call("GET", GetURL.String(), hdr, nil)

		if err != nil {
			loggers.ErrorLogger().Major("%v", err)
			return err
		}

		switch configType {
		case decision:
			Msg := &DecisionConfigRespData{}

			err = json.Unmarshal(respData, Msg)
			if err != nil {
				return err
			}

			DecisionStoragePush(Msg.ConfId, Msg.Configuration)
			loggers.InfoLogger().Comment("Save Decision StoragePush Data")
			break

		case smsf:
			Msg := &CommonConfigRespData{}

			err = json.Unmarshal(respData, Msg)
			if err != nil {
				loggers.ErrorLogger().Major("Unmarshl common config")
				return err
			}
			loggers.InfoLogger().Comment("Save CommonStoragePush Data")
			CommonStoragePush(Msg.ConfId, Msg.Configuration)
			break

		case smsc:
			Msg := &SmscConfigRespData{}
			err = json.Unmarshal(respData, Msg)
			if err != nil {
				loggers.ErrorLogger().Major("Unmarshl SMSC config")
				return err
			}

			loggers.InfoLogger().Comment("Save SMSCStoragePush Data")
			SmscStoragePush(Msg.ConfId, Msg.Configuration)
			break

		default:
			loggers.ErrorLogger().Major("Invalid Configuration ID : %s", confId)
			return errors.New(fmt.Sprintf("Invalid Configuration ID : %s", confId))

		}

		loggers.InfoLogger().Comment("Configuration Count Result Code : %d", GetResp.StatusCode)
	}

	return err
}

func (c *ConfigClient) SubscribeToUccmsWatch() {
	loggers.InfoLogger().Comment("Initial Watch POST Start..")
	watchResp := &WatchCtrlResData{}

	// Get My IP for callbackUrl
	myIp := getMyIp()

	// To do All Get ConfId+CallbackURl(To do /test/watch Path..Current Path Fix)
	_, configIdMap := GetConfIdStorage()

	//for i := 0; i < count; i++ {
	for _, confId := range configIdMap {
		loggers.InfoLogger().Comment("confId : %s", confId)

		callback := fmt.Sprintf("%s:%s/test/watch", myIp, c.ConfigPort)
		loggers.InfoLogger().Comment("CallBack url : %s", callback)

		reqbody := WatchCtrlReqData{
			ConfId:   confId,
			CallBack: callback,
		}

		jsonbody, err := json.Marshal(reqbody)
		if err != nil {
			loggers.ErrorLogger().Major("json.Mashal Err : %s", err.Error())
			return
		}

		PostURL := "/app/v1/watch"

		hdr := http.Header{}
		hdr.Add("accept", "application/json")
		hdr.Add("Content-Type", "application/json")

		if c.scheme != "http" {
			PostResp, err := c.router.SendRequest(context.Background(), PostURL, "POST", hdr, jsonbody, 2*time.Second)
			if err != nil {
				loggers.ErrorLogger().Major("WATCH PostResp Response err() : %s ", err.Error())
				return
			}

			err = json.Unmarshal(PostResp.Response.([]byte), watchResp)
			if err != nil {
				return
			}
			SetWatchIdStorage(watchResp.WatchId)
			loggers.InfoLogger().Comment("Result Code : %d, Body : %s", PostResp.StatusCode, string(PostResp.Response.([]byte)))

		} else {

			PostResp, PostData, err := c.cli.Call(http.MethodPost, PostURL, hdr, jsonbody)
			if err != nil {
				loggers.ErrorLogger().Major("WATCH PostResp Response err() : %s ", err.Error())
				return
			}

			err = json.Unmarshal(PostData, watchResp)
			if err != nil {
				return
			}
			SetWatchIdStorage(watchResp.WatchId)

			loggers.InfoLogger().Comment("Result Code : %d, Body : %s", PostResp.StatusCode, string(PostData))

			//need to unmarshal for Resp. data from UCMMS

		}

	}
}
