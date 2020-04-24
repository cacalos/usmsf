package common

import (
	"encoding/json"
	"fmt"
	"net/http"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	//	"github.com/gin-gonic/gin"
	"camel.uangel.com/ua5g/usmsf.git/msg5g"
	"github.com/labstack/echo"
)

//HTTPServer HTTP 기반 서비스 서버의 공통 객체
type HTTPServer struct {
	Addr string
	//	Handler            *gin.Engine
	Handler            *echo.Echo
	authRequired       bool     //Authenticiation이 요구 되는지 여부
	authCredentials    [][]byte // slice with base64-encoded credentials
	probeResistDomain  string
	probeResistEnabled bool
}

type Header struct {
	Cause string `json:"cause"`
}

// Errorf 전달된 이름의 Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) Errorf(code int, title string, formatstr string, args ...interface{}) *msg5g.ProblemDetails {
	err := fmt.Errorf(formatstr, args...)
	pd := &msg5g.ProblemDetails{
		Title:  title,
		Status: code,
		Detail: err.Error(),
	}
	return pd
}

// ErrorNotFound 전달된 Resource를 찾을 수 없다.
func (s *HTTPServer) ErrorNotFound(ctx echo.Context) *msg5g.ProblemDetails {
	return s.Errorf(404, "Not Found", "'%v' is not found.", ctx.Request().URL.String())
}

// ErrorBadRequest 전달된 Request가 문제가 있다.
func (s *HTTPServer) ErrorBadRequest(ctx echo.Context) *msg5g.ProblemDetails {
	return s.Errorf(400, "Bad Request", "'%v' is bad request.", ctx.Request().URL.String())
}

// ErrorInvalidProtocol 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorInvalidProtocol(ctx echo.Context) *msg5g.ProblemDetails {
	return s.Errorf(400, "Bad Request", "'%v %v' is sent on invalid protocol.", ctx.Request().Method, ctx.Request().URL.String())
}

// ErrorUnauthorized 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorUnauthorized(ctx echo.Context, sender, message string) *msg5g.ProblemDetails {
	return s.Errorf(401, "Unauthorized", "'%v' is unauthorized.%v", sender, message)
}

// ErrorForbidden 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorForbidden(ctx echo.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(403, "Forbidden", "'%v'.%v", ctx.Request().URL.String(), message)
}

// ErrorNotAllowed 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) ErrorNotAllowed(ctx echo.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(405, "NotAllowed", "'%v'.%v", ctx.Request().URL.String(), message)
}

// ErrorServiceUnavailable 서비스를 이용할 수 없다.
func (s *HTTPServer) ErrorServiceUnavailable(ctx echo.Context, message string) *msg5g.ProblemDetails {
	return s.Errorf(503, "ServiceUnavailable", "'%v'.%v", ctx.Request().URL.String(), message)
}

// RespondError 전달된 에러를 응답으로 반환한다.
func (s *HTTPServer) RespondError(ctx echo.Context, err error) error {
	var pd *msg5g.ProblemDetails
	switch v := err.(type) {
	case *msg5g.ProblemDetails:
		pd = v
	case *errcode.ErrorWithCode:
		pd = &msg5g.ProblemDetails{
			Title:  v.Title,
			Status: v.Code,
			Detail: v.Error(),
		}
	default:
		pd = &msg5g.ProblemDetails{
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: v.Error(),
		}
	}

	return s.RespondProblemDetails(ctx, pd)
}

// RespondProblemDetails 전달된 Problem Details 에러를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondProblemDetails(ctx echo.Context, pd *msg5g.ProblemDetails) error {
	rbody, err := json.Marshal(pd)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
	} else {
		//		ctx.Data(pd.Status, "application/problem+json", rbody)
		//ctx.JSONBlob(pd.Status, "application/problem+json", rbody)
		ctx.Blob(pd.Status, "application/problem+json", rbody)
	}
	return pd
}

// RespondParseError JSON Parsing Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondParseError(ctx echo.Context, err error) error {
	pd := msg5g.JSONParseError(ctx.Request().URL.String(), err)
	return s.RespondProblemDetails(ctx, pd)
}

// RespondSystemError System Error가 발생했을 때 해당 서비스에 대한 Error를 HTTP 응답으로 반환한다.
func (s *HTTPServer) RespondSystemError(ctx echo.Context, err error) error {
	pd := msg5g.SystemError(ctx.Request().URL.String(), err)
	return s.RespondProblemDetails(ctx, pd)
}

// RespondNotFound 전달된 Resource를 찾을 수 없다.
func (s *HTTPServer) RespondNotFound(ctx echo.Context) error {
	return s.RespondProblemDetails(ctx, s.ErrorNotFound(ctx))
}

// RespondBadRequest 전달된 Request가 문제가 있다.
func (s *HTTPServer) RespondBadRequest(ctx echo.Context) error {
	return s.RespondProblemDetails(ctx, s.ErrorBadRequest(ctx))
}

// RespondBadRequest 전달된 Request가 문제가 있다.
func (s *HTTPServer) RespondNotAllowed(ctx echo.Context, message string) error {
	return s.RespondProblemDetails(ctx, s.ErrorNotAllowed(ctx, message))
}

// RespondInvalidProtocol 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) RespondInvalidProtocol(ctx echo.Context) error {
	return s.RespondProblemDetails(ctx, s.ErrorInvalidProtocol(ctx))
}

// RespondUnauthorized 잘못된 프로토콜로 메시지가 전달 되었다.
func (s *HTTPServer) RespondUnauthorized(ctx echo.Context, sender, message string) error {
	return s.RespondProblemDetails(ctx, s.ErrorUnauthorized(ctx, sender, message))
}

// RespondForbiddend 서비스를 access할 수 없다.
func (s *HTTPServer) RespondForbidden(ctx echo.Context, message string) error {
	return s.RespondProblemDetails(ctx, s.ErrorForbidden(ctx, message))
}

// RespondForbiddend 서비스를 access할 수 없다.
func (s *HTTPServer) RespondServiceUnavailable(ctx echo.Context, message string) error {
	return s.RespondProblemDetails(ctx, s.ErrorServiceUnavailable(ctx, message))
}
