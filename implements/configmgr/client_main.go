package configmgr

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/implements/db"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

var loggers = common.SamsungLoggers()

type ConfigClient struct {
	router uclient.HTTPRouter
	cli    *common.HTTPClient

	cliConf common.HTTPCliConf
	useFlag string
	scheme  string

	httpsAddr  string
	ConfigPort string
	uccmshost  string

	decisionPath string
	smsfPath     string
	smscPath     string

	udecisionPath string
	usmsfPath     string
	usmscPath     string
}

func NewConfigClient(
	cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
	httpcli uclient.HTTP,
) *ConfigClient {

	c := &ConfigClient{}

	err := c.LoadConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL LOAD -> CONFIG(ConfigMgr)")
		return nil
	}

	err = c.uccmshostConfig(httpcli)
	if err != nil {
		loggers.ErrorLogger().Major("FAIL LOAD -> uccmshostConfig(ConfigMgr)")
		return nil
	}

	if c.useFlag == "svc" {
		dbmgr := db.UCCMSMariaNew(cfg)
		if dbmgr == nil {
			loggers.ErrorLogger().Major("FAIL Connect Uccms -> CONFIG(UCCMSMariaNew)")
			return nil
		}

		uccms := db.NewMariaDbSubInfoDAO(dbmgr)
		if uccms == nil {
			loggers.ErrorLogger().Major("FAIL Connect Uccms -> CONFIG(NewMariaDbSubInfoDAO)")
			return nil
		}

		err = uccms.UccmsDelete()
		if err != nil {
			loggers.ErrorLogger().Major("FAIL Connect Uccms -> CONFIG(Delete)")
			return nil
		}
		c.DeleteAllConfigData()
		c.DeleteAllmetaData()
		c.PushAllMetaData()
		c.PushAllConfigData()
	}

	c.GetAllConfigFromUCCMS()

	// Initial Watch All POST
	c.SubscribeToUccmsWatch()

	return c
}

func (c *ConfigClient) LoadConfig(cfg uconf.Config) (err error) {

	smsfConf := cfg.GetConfig("http-configmgr")

	httpConf := cfg.GetConfig("http-configmgr.http")
	httpPort := httpConf.GetInt("port", 8090)
	c.ConfigPort = strconv.Itoa(httpPort)

	if smsfConf != nil {

		c.cliConf.DialTimeout = smsfConf.GetDuration("map-client.connection.timeout", time.Second*20)
		c.cliConf.DialKeepAlive = smsfConf.GetDuration("map-client.connection.keep-alive", time.Second*20)
		c.cliConf.IdleConnTimeout = smsfConf.GetDuration("map-client.connection.expire-time", 2*time.Minute)
		c.cliConf.InsecureSkipVerify = true

		c.useFlag = smsfConf.GetString("use", "map")
		c.scheme = smsfConf.GetString("scheme", "http")

		c.decisionPath = smsfConf.GetString("decisionPath", "")
		c.smsfPath = smsfConf.GetString("smsfPath", "")
		c.smscPath = smsfConf.GetString("smscPath", "")

		c.udecisionPath = smsfConf.GetString("udecisionPath", "")
		c.usmsfPath = smsfConf.GetString("usmsfPath", "")
		c.usmscPath = smsfConf.GetString("usmscPath", "")

	} else {
		return errors.New("Init Fail : HTTP CONFIG(Svc)")

	}
	return nil
}

func (c *ConfigClient) uccmshostConfig(httpcli uclient.HTTP) (err error) {

	c.uccmshost = os.Getenv("CONF_POD_HOST")

	loggers.InfoLogger().Comment("Get Env : %s", c.uccmshost)

	if c.uccmshost != "" {
		if c.scheme != "" {
			if c.scheme != "http" {
				c.router = uclient.HTTPRouter{
					//	Scheme:           "http",
					Servers: []string{c.uccmshost},
					Client:  httpcli,
				}
			} else {
				c.cli, err = common.NewHTTPClient(&c.cliConf, "http", c.uccmshost, c.uccmshost, 1, nil)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return errors.New("Set FAil : uccmshost Config")
	}

	return nil
}

func (c *ConfigClient) GetAllConfigFromUCCMS() {
	var err error

	if c.useFlag == "map" { //dia pod
	RetrySmsf:
		err = c.GetCommonConfig()
		if err != nil {
			time.Sleep(1 * time.Second)
			loggers.ErrorLogger().Major("GetDataFail From UCCMS SMSFConfig: %v", err)
			goto RetrySmsf
		}

	RetrySmsc:
		err = c.GetSmscConfig()
		if err != nil {
			time.Sleep(1 * time.Second)
			loggers.ErrorLogger().Major("GetDataFail From UCCMS SMSCConfig: %v", err)
			goto RetrySmsc
		}
	} else { // svc pod
	RetryDecision:
		err = c.GetDecisionConfig()
		if err != nil {
			time.Sleep(1 * time.Second)
			loggers.ErrorLogger().Major("GetDataFail From UCCMS Decision : %v", err)
			goto RetryDecision
		}
	}
}

func (c *ConfigClient) PushAllMetaData() {

	data := PushMyMetaData(c.decisionPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.decisionPath)
		return
	} else {
		c.SendReq(http.MethodPost, "", "", data)
	}

	data = PushMyMetaData(c.smscPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.smscPath)
		return
	} else {
		c.SendReq(http.MethodPost, "", "", data)
	}

	data = PushMyMetaData(c.smsfPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.decisionPath)
		return
	} else {
		c.SendReq(http.MethodPost, "", "", data)
	}

}

func (c *ConfigClient) DeleteAllmetaData() {

	url := "SMSFDecisionConfiguration_keys"
	c.SendReq(http.MethodDelete, "", url, nil)

	url = "SMSFSmscConfiguration_keys"
	c.SendReq(http.MethodDelete, "", url, nil)

	url = "SMSFSmsfConfiguration_keys"
	c.SendReq(http.MethodDelete, "", url, nil)
}

func (c *ConfigClient) DeleteAllConfigData() {

	url := "SMSFDecisionConfiguration_keys_v1.0"
	c.confSendReq(http.MethodDelete, "", url, nil)

	url = "SMSFSmscConfiguration_keys_v1.0"
	c.confSendReq(http.MethodDelete, "", url, nil)

	url = "SMSFSmsfConfiguration_keys_v1.0"
	c.confSendReq(http.MethodDelete, "", url, nil)
}

func (c *ConfigClient) PushAllConfigData() {
	data := PushMyConfigData(c.udecisionPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.decisionPath)
		return
	} else {

		url := "SMSFDecisionConfiguration_keys_v1.0"
		c.confSendReq(http.MethodPatch, "", url, data)
	}
	data = PushMyConfigData(c.usmscPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.smscPath)
		return
	} else {
		url := "SMSFSmscConfiguration_keys_v1.0"
		c.confSendReq(http.MethodPatch, "", url, data)
	}

	data = PushMyConfigData(c.usmsfPath)
	if data == nil {
		loggers.ErrorLogger().Major("FAIL Insert MetaData to UCCMS\n%s\n", c.decisionPath)
		return
	} else {
		url := "SMSFSmsfConfiguration_keys_v1.0"
		c.confSendReq(http.MethodPatch, "", url, data)
	}
}
