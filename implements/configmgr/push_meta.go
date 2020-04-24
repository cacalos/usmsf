package configmgr

import (
	"bytes"
	//	"camel.uangel.com/ua5g/ulib.git/hocon"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
)

func PushMyMetaData(path string) []byte {

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

	loggers.InfoLogger().Comment("Req. To UCCMS (Dicision MetaDAta) : %s", string(filedata))
	return filedata

}

func (c *ConfigClient) SendReq(method string, id string, query string, body []byte) {
	var baseURL string
	if query == "" {
		baseURL = fmt.Sprintf("/app/v1/metadata")
	} else {
		baseURL = fmt.Sprintf("/app/v1/metadata/%s", query)
	}

	rselecPostURL, err := url.Parse(baseURL)
	if err != nil {
		loggers.ErrorLogger().Major("Malformed URL : %s", err.Error())
		return
	}

	hdr := http.Header{}
	hdr.Add("accept", "application/json")
	hdr.Add("Content-Type", "application/json")

	if c.scheme != "http" {
		GetResp, err := c.router.SendRequest(context.Background(),
			rselecPostURL.String(), method, hdr, body, 2*time.Second)
		if err != nil {
			loggers.ErrorLogger().Major("failed to send request -> url(method : %s) : %s%s", method, c.uccmshost, rselecPostURL.String())
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
		GetResp, respData, err := c.cli.Call(method, rselecPostURL.String(), hdr, body)

		if err != nil {
			loggers.ErrorLogger().Major("failed to send request -> url(method : %s) : %s%s", method, c.uccmshost, rselecPostURL.String())
			return
		} else {

			loggers.InfoLogger().Comment("Result:         %d", GetResp.StatusCode)

			var i interface{}
			json.Unmarshal(respData, &i)

			var out bytes.Buffer

			enc := json.NewEncoder(&out)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			err := enc.Encode(i)
			if err == nil {
				out.WriteTo(os.Stdout)
			}
		}
	}
}
