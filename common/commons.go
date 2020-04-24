package common

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/usmsf.git/implements/utils"
)

// Forwarder 공통 정의 및 사용 함수들
const SMSF_BOUNDARY = "SMSF_Boundary"

var _hopByHopHeaders = []string{
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Upgrade",
	"Connection",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
}

// CopyHeader 전달된 Header를 복사한다.
func CopyHeader(dst, src http.Header, isProxy bool) {
	for header, values := range src {
		for _, value := range values {
			dst.Add(header, value)
		}
	}
	if isProxy {
		RemoveHopByHopHeader(dst)
	}
}

// RemoveHopByHopHeader Hop by Hop Header를 삭제한다.
func RemoveHopByHopHeader(header http.Header) {
	connectionHeaders := header.Get("Connection")
	for _, h := range strings.Split(connectionHeaders, ",") {
		header.Del(strings.TrimSpace(h))
	}
	for _, h := range _hopByHopHeaders {
		header.Del(h)
	}
}

/*
// RemoveTraceHeader Trace용 Header를 삭제한다.
func RemoveTraceHeader(header http.Header) {
	header.Del(tracemgr.HeaderNames.RequestID)
	header.Del(tracemgr.HeaderNames.TraceID)
	header.Del(tracemgr.HeaderNames.SpanID)
	header.Del(tracemgr.HeaderNames.ParentSpanID)
	header.Del(tracemgr.HeaderNames.Sampled)
	header.Del(tracemgr.HeaderNames.Flags)
	header.Del(tracemgr.HeaderNames.OtSpanContext)
	header.Del(tracemgr.HeaderNames.TraceKey)
}
*/

// PrepareProxyHTTPRequest Proxy로 보낼 메시지를 조정한다.
func PrepareProxyHTTPRequest(req *http.Request, hideIP, hideVia bool) error {
	RemoveHopByHopHeader(req.Header)
	if !hideIP {
		req.Header.Add("Forwarded", "for=\""+req.RemoteAddr+"\"")
	}
	if !hideVia {
		// https://tools.ietf.org/html/rfc7230#section-5.7.1
		req.Header.Add("Via", strconv.Itoa(req.ProtoMajor)+"."+strconv.Itoa(req.ProtoMinor)+" usmsf")
	}
	if req.Body == nil {
		return nil
	}
	if req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS" || req.Method == "TRACE" {
		//본문이 없는 메시지 일지라도 body를 모두 읽을 필요가 있음.
		reqbuf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return errcode.Errorf(http.StatusBadRequest, "Bad Request", "failed to read request Body: "+err.Error())
		}
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(reqbuf)), nil
		}
		req.Body, _ = req.GetBody()
	}

	return nil
}

// SendResponse 전달된 Response를 Response Writer를 통해 송신한다.
func SendResponse(rw http.ResponseWriter, rsp *http.Response, isProxy, hideVia bool, flushInterval time.Duration) {
	// Server 헤더 삭제 후 Via 헤더 추가
	rw.Header().Del("Server")
	if isProxy {
		if !hideVia {
			rw.Header().Add("Via", strconv.Itoa(rsp.ProtoMajor)+"."+strconv.Itoa(rsp.ProtoMinor)+" usmsf")
		}
	}
	CopyHeader(rw.Header(), rsp.Header, isProxy)
	rw.WriteHeader(rsp.StatusCode)
	if rsp.Body != nil {
		utils.CopyIOWithFlush(rw, rsp.Body, flushInterval)
	}
}

func IsContain(strs []string, str string) bool {
	for i := range strs {
		if strs[i] == str {
			return true
		}
	}
	return false
}
