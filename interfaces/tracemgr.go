package interfaces

import (
	"context"
	"net/http"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/savsgio/atreugo/v7"
)

type TraceMgr interface {
	Tracer() opentracing.Tracer
	Level() ulog.LogLevel

	GinHTTPTraceHandler(serviceName string, level ulog.LogLevel) gin.HandlerFunc
	AtreugoHTTPTraceHandler(serviceName string, level ulog.LogLevel) atreugo.Middleware

	StartSpanFromServerHTTPReq(req *http.Request, serviceName string, level ulog.LogLevel) (opentracing.Span, *http.Request)
	StartSpanFromClientHTTPReq(req *http.Request, serviceName string, level ulog.LogLevel) (opentracing.Span, *http.Request)

	StartSpanFromServerHTTPHdr(headers http.Header, serviceName string, level ulog.LogLevel) (opentracing.Span, context.Context)
	StartSpanFromClientHTTPHdr(headers http.Header, serviceName string, level ulog.LogLevel) (opentracing.Span, context.Context)
	StartSpanFromHTTPHdr(headers http.Header, serviceName string) (opentracing.Span, context.Context)
	StartSpanFromContext(traceContext context.Context, serviceName string) (opentracing.Span, context.Context)

	SpanFromHTTPReq(req *http.Request) opentracing.Span
	SpanFromContext(traceContext context.Context) opentracing.Span

	InjectToHTTP(traceContext context.Context, headers http.Header) error
	GetTraceKey(traceContext context.Context) string
	SetTraceKey(traceContext context.Context, traceKey string) context.Context

	LogFields(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsPanic(ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsFatal(ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsError(ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsWarn(ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsInfo(ctx interface{}, serviceName string, keyValues ...interface{})
	LogFieldsDebug(ctx interface{}, serviceName string, keyValues ...interface{})

	SetTags(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsPanic(ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsFatal(ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsError(ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsWarn(ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsInfo(ctx interface{}, serviceName string, keyValues ...interface{})
	SetTagsDebug(ctx interface{}, serviceName string, keyValues ...interface{})

	LogHTTPRes(traceContext context.Context, res *http.Response, serviceName string, level ulog.LogLevel)

	TraceHeaderName() string
}
