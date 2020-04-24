package common

import (
	"strconv"

	"path/filepath"
	"strings"

	"os"

	"camel.uangel.com/ua5g/ulib.git/testhelper"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	//"camel.uangel.com/ua5g/ulib.git/ulog"
)

/*
configuration 변경 시 다시 로드 되도록 작성된 configuration 정보
TRACER.Path
SERVICE.OAuthTokenValidTime
SERVICE.OAuthTokenIssuer
*/

var cfg uconf.Config

// DefaultConfig 시험을 위한 기본 Configuration
func DefaultConfig() uconf.Config {
	if cfg == nil {
		cfg = NewConfig("unrf.toml")
	}

	return cfg
}

// NewConfig 전달된 파일이름의 새로운 configuration 파일을 반환한다.
func NewConfig(fname string) uconf.Config {
	var path string
	if !strings.Contains(fname, "/") {
		if IsExistFile("./" + fname) {
			path = "./" + fname
		} else if os.Getenv("UASYS_HOME") != "" {
			path = os.Getenv("UASYS_HOME") + "/data/" + fname
		} else {
			path = GetModuleRootPath("common") + "/resources/" + fname
		}
	} else {
		path = fname
	}
	cfg := testhelper.LoadConfigFromFile(filepath.FromSlash(path))
	return cfg
}

func GetNrfURL(cfg uconf.Config) string {
	nrfHost := os.Getenv("NRF_HOST")
	url := cfg.GetString("url", "")
	scheme := cfg.GetString("http.scheme", "http")
	host := cfg.GetString("http.host", "localhost:8080")
	port := cfg.GetInt("http.port", 8080)
	if nrfHost != "" {
		if scheme == "http" {
			if port != 0 && port != 80 {
				return "http://" + nrfHost + ":" + strconv.Itoa(port)
			}
		} else if scheme == "https" {
			if port != 0 && port != 443 {
				return "https://" + nrfHost + ":" + strconv.Itoa(port)
			}
		}
		return scheme + "://" + nrfHost
	}
	if url == "" {
		return scheme + "://" + host
	}
	return url
}

// GetCacheURL Cache Server의 URL 을 반환한다.
func GetCacheURL(cfg uconf.Config) string {
	cacheHost := os.Getenv("NRF_CACHE_HOST")
	if cacheHost != "" {
		return "http://" + cacheHost
	}

	cacheHost = cfg.GetString("cache-server.http.host", "")
	if cacheHost != "" {
		return "http://" + cacheHost
	}

	addr := cfg.GetString("cache-server.http.address", "")
	if addr == "" {
		addr = "localhost"
	}

	port := cfg.GetInt("cache-server.http.port", 8080)
	if port != 0 && port != 80 {
		return "http://" + addr + ":" + strconv.Itoa(port)
	}

	return "http://" + addr
}
