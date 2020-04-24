package configmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"

	//"net/url"
	"os"
	"path/filepath"
	"time"
)

func PushMyConfigData(path string) []byte {

	filedata, err := ioutil.ReadFile(path)
	if err != nil {
		loggers.ErrorLogger().Major("invalid status file(filename:%s)\n", path)
		return nil
	}

	if filepath.Ext(path) == ".yaml" {
		loggers.InfoLogger().Comment("yaml stream\n %s\n", string(filedata))
		jsondata, _ := yaml.YAMLToJSON(filedata)
		loggers.InfoLogger().Comment("json stream\n %s\n", string(jsondata))
		return jsondata
	}

	return filedata

}

func (c *ConfigClient) confSendReq(method string,
	id string,
	query string,
	body []byte,
) {
	var baseURL string
	baseURL = fmt.Sprintf("/app/v1/configurations/%s", query)

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	loggers.InfoLogger().Comment("Body(to : %s >>>> %s%s", string(body), c.cli.RootPath, baseURL)
	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(), baseURL, method, hdr, body, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("failed to send request -> url(method : %s) : %s%s", method, c.uccmshost, baseURL)
			return
		} else {

			//	err, code, resbyte := client.SendReq(method, url, body)
			loggers.InfoLogger().Comment("Result:         ", GetResp.StatusCode)

			var i interface{}
			json.Unmarshal(GetResp.ResponseBytes(), &i)
			//		jsonres, _ := hocon.JSONStringify(i)

			var out bytes.Buffer
			fmt.Println("Response: ")

			enc := json.NewEncoder(&out)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			err := enc.Encode(i)
			if err == nil {
				out.WriteTo(os.Stdout)
			}

		}
	} else {

		loggers.InfoLogger().Comment("Request URI : %s - %s%s", method, c.cli.RootPath, baseURL)
		GetResp, respData, err := c.cli.Call(method, baseURL, hdr, body)
		if err != nil {
			loggers.ErrorLogger().Major("failed to send request -> url(method : %s) : %s%s",
				method, c.cli.RootPath, baseURL)
			return
		}

		if GetResp.StatusCode > 300 {
			loggers.ErrorLogger().Major("StatusCode from UCCMS : %s", GetResp.StatusCode)
			return
		}
		loggers.InfoLogger().Comment("Result:         %d", GetResp.StatusCode)

		var i interface{}
		json.Unmarshal(respData, &i)
		var out bytes.Buffer

		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		err = enc.Encode(i)
		if err == nil {
			out.WriteTo(os.Stdout)
		}

	}
}
