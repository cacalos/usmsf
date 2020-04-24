package common

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	jd "github.com/josephburnett/jd/lib"
)

// StrHash 전달된 string 값에 대한 Hash 값을 반환한다.
func StrHash(s string) uint32 {
	algorithm := fnv.New32a()
	algorithm.Write([]byte(s))
	return algorithm.Sum32()
}

// GetMicroSecTime 전달된 시간의 Microsecond 단위 시간을 반환한다.
func GetMicroSecTime(t time.Time) time.Time {
	return time.Unix(0, (t.UnixNano()/int64(time.Microsecond))*int64(time.Microsecond))
}

// GetNowMicroSecTime 현재 시간의 Microsecond 단위 시간을 반환한다.
func GetNowMicroSecTime() time.Time {
	t := time.Now()
	return time.Unix(0, (t.UnixNano()/int64(time.Microsecond))*int64(time.Microsecond))
}

// GetShardID 전달된 ID의 Sharding Actor 정보를 반환한다.
func GetShardID(id string, shardCount int) string {
	i := uint64(StrHash(id) % uint32(shardCount))
	return strconv.FormatUint(i, 10)
}

// CreateHTTPClient http.Client 인스턴스를 반환한다.
func CreateHTTPClient() *http.Client {
	keepAliveTimeout := 600 * time.Second
	timeout := 2 * time.Second
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: keepAliveTimeout,
		}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	return &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
}

// GetValueFromStringPtr 문자열 포인터 변수로부터 값을 가져 옴. nil 인 경우, 빈 문자열을 반환.
func GetValueFromStringPtr(src *string) string {
	result := ""
	if src != nil && len(*src) > 0 {
		result = *src
	}
	return result
}

// parameter values for StringSliceContains() function
const (
	StringSliceContainsCaseInsensitive = 0
	StringSliceContainsCaseSensitive   = 1
)

// StringSliceContains 문자열 슬라이스에 지정한 문자열이 존재하는 지 확인
func StringSliceContains(slice []string, value string, caseSensitiveOption int) bool {
	for _, s := range slice {
		if caseSensitiveOption == StringSliceContainsCaseInsensitive {
			if strings.ToUpper(s) == strings.ToUpper(value) {
				return true
			}
		} else {
			if s == value {
				return true
			}
		}
	}
	return false
}

// GetStringKVMap key=value[,key=value] 형식의 string 값을 map으로 변환해 반환한다.
func GetStringKVMap(src string) (map[string]string, error) {
	rslt := make(map[string]string)
	sval := ""
	key := ""
	cur := 0
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\t':
		case '\n':
		case ' ':
		case '=':
			key = src[cur:i]
			key = strings.Trim(key, "\t\n ")
			if sval != "" && key != "" {
				return nil, fmt.Errorf("Invalid key format. (pos=%v..%v)", cur, i)
			}
			if key == "" {
				if sval == "" {
					return nil, fmt.Errorf("key is empty. (pos=%v..%v)", cur, i)
				}
				key = sval[1 : len(sval)-1]
				sval = ""
			}
			cur = i + 1
		case '"':
			value := src[cur:i]
			value = strings.Trim(value, "\t\n ")
			if value != "" {
				return nil, fmt.Errorf("Invalid string format. (pos=%v..%v)", cur, i)
			}
			cur = i
			idx := strings.Index(src[cur+1:], `"`)
			if idx < 0 {
				return nil, fmt.Errorf("invalid string format. (pos=%v..%v)", cur, len(src))
			}
			i = i + idx + 1
			v := src[cur : i+1]
			if sval != "" {
				return nil, fmt.Errorf("Invalid string defines. (pos=%v..%v)", cur, i)
			}
			sval = v
			cur = i + 1
		case ',':
			value := src[cur:i]
			value = strings.Trim(value, "\t\n ")
			if sval != "" && value != "" {
				return nil, fmt.Errorf("Invalid value format. (pos=%v..%v)", cur, i)
			}
			if value == "" && sval != "" {
				value = sval[1 : len(sval)-1]
				sval = ""
			}
			if key != "" {
				rslt[key] = value
				key = ""
			}
			cur = i + 1
		}
	}
	if cur <= len(src) {
		value := src[cur:]
		value = strings.Trim(value, "\t\n ")
		if sval != "" && value != "" {
			return nil, fmt.Errorf("Invalid value format. (pos=%v..%v)", cur, len(src))
		}
		if value == "" && sval != "" {
			value = sval[1 : len(sval)-1]
			sval = ""
		}
		if key != "" {
			rslt[key] = value
		}
	}

	return rslt, nil
}

///////////////////////////////////////////////////////////////////////////////
// github.com/josephburnett/jd/lib 기반의 Diff Tool

// Diff 두 구조체를 비교하여, 구조체의 변경 된 애트리뷰트(들)에 대한 JSON-Path 배열을 반환한다.
func Diff(src interface{}, with interface{}) ([]string, error) {
	srcBytes, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	srcJSON, err := jd.ReadJsonString(string(srcBytes))
	if err != nil {
		return nil, err
	}

	withBytes, err := json.Marshal(with)
	if err != nil {
		return nil, err
	}
	withJSON, err := jd.ReadJsonString(string(withBytes))
	if err != nil {
		return nil, err
	}

	diff := srcJSON.Diff(withJSON)
	return getChangedPaths(diff), nil
}

func getChangedPaths(diff jd.Diff) []string {
	paths := []string{}
	for _, element := range diff {
		pathcomponents := []string{}
		for _, p := range element.Path {
			v := strings.Replace(p.Json(), "\"", "", -1)
			pathcomponents = append(pathcomponents, v)
		}
		paths = append(paths, fmt.Sprintf("/%s", strings.Join(pathcomponents, "/")))
	}
	return paths
}

// JSONPathContains JSON-path 내에 지정한 서브 경로명이 포함되는지를 반환 (subPaths 슬라이스에 지정하는 경로명에는 '/'를 포함하지 않아야 함.)
func JSONPathContains(path string, subPaths []string) bool {
	pathElements := strings.Split(path, "/")
	if len(pathElements) == 0 {
		return false
	}
	for _, p := range pathElements {
		for _, subPath := range subPaths {
			if p == subPath {
				return true
			}
		}
	}
	return false
}

// github.com/josephburnett/jd/lib 기반의 Diff Tool
///////////////////////////////////////////////////////////////////////////////

// GetPrivateKeyBytesFromFile PEM 형식의 private key 파일을 읽어 private key 값을 구하고 바이트 배열로 반환
func GetPrivateKeyBytesFromFile(keyfilePath string, rsaPrivateKeyPassword string) ([]byte, error) {
	raw, err := ioutil.ReadFile(keyfilePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(raw)
	if block.Type != "PRIVATE KEY" && block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("it's not a private key file: %s", keyfilePath)
	}

	var blockBytes []byte
	if rsaPrivateKeyPassword != "" {
		blockBytes, err = x509.DecryptPEMBlock(block, []byte(rsaPrivateKeyPassword))
	} else {
		blockBytes = block.Bytes
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(blockBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(blockBytes); err != nil {
			return nil, fmt.Errorf("unable to parse private key: %s", err)
		}
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not RSA private key")
	}
	return privateKey.N.Bytes(), nil
}

// GetPublicKeyBytesFromCert PEM 형식의 Certificate 파일을 읽어 public key 값을 구하고 바이트 배열로 반환
func GetPublicKeyBytesFromCert(certfilePath string, certificatePassword string) ([]byte, error) {
	raw, err := ioutil.ReadFile(certfilePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(raw)
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("it's not a certificate file: %s", certfilePath)
	}

	var blockBytes []byte
	if certificatePassword != "" {
		blockBytes, err = x509.DecryptPEMBlock(block, []byte(certificatePassword))
	} else {
		blockBytes = block.Bytes
	}

	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(blockBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate: %s", err)
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not RSA public key")
	}
	return publicKey.N.Bytes(), nil
}
