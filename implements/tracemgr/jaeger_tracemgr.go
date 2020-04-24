package tracemgr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"camel.uangel.com/ua5g/ulib.git/testhelper"

	"camel.uangel.com/ua5g/ulib.git/errcode"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"camel.uangel.com/ua5g/ulib.git/ulog"
	"camel.uangel.com/ua5g/ulib.git/watcher"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"

	"github.com/gin-gonic/gin"
	"github.com/savsgio/atreugo/v7"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"
)

var loggers = common.SamsungLoggers()

// TODO hanmouse: delete
//var logger = ulog.GetLogger("com.uangel.usmsf.tracemgr")
var jaegerLogger = ulog.AsJaegerLogger(ulog.GetLogger("com.github.uber.jaeger"))

var HeaderNames = struct {
	RequestID     string
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Sampled       string
	Flags         string
	OtSpanContext string
	TraceKey      string
}{
	RequestID:     "x-request-id",
	TraceID:       "Uber-Trace-Id",
	SpanID:        "x-b3-spanid",
	ParentSpanID:  "x-b3-parentspanid",
	Sampled:       "x-b3-sampled",
	Flags:         "x-b3-flags",
	OtSpanContext: "x-ot-span-context",
	TraceKey:      "xx-trace-key",
}

type contextKey struct{}

var traceContextKey = contextKey{}

const traceContextKeyStr = "traceContext"

type JaegerTraceMgr struct {
	tracer      opentracing.Tracer
	closer      io.Closer
	level       ulog.LogLevel
	mode        string
	logHTTPBody bool
	traceKeyMap map[string]bool
	fileWatcher watcher.File
	configPath  string
	defaultConf jaegerConfig.Configuration
	traceConf   uconf.Config
	//	traceInfo   *svctracemgr.TraceSvcPod
}

type TraceConfig struct {
	Level       string `yaml:"level"`
	Mode        string `yaml:"mode"`
	LogHTTPBody bool   `yaml:"logHttpBody"`
}

//func NewJaegerTraceMgr(cfg uconf.Config, fileWatcher watcher.File, lf ulog.LoggerFactory, traceInfo *controller.TraceSvcPod) interfaces.TraceMgr {
func NewJaegerTraceMgr(cfg uconf.Config, fileWatcher watcher.File, lf ulog.LoggerFactory) interfaces.TraceMgr {

	configPath := cfg.GetString("trace.config-path", "./jaegertrace")
	serviceName := cfg.GetString("trace.default-service-name", "USMSF")
	/*
		level := cfg.GetString("trace.level", "warn")
		mode := cfg.GetString("trace.mode", "all")
		logHTTPBody := cfg.GetBoolean("trace.log-http-body", true)
		traceKeyArray := cfg.GetStringArray("trace.key")
	*/
	traceKeyArray := cfg.GetStringArray("trace.key")

	loggers.InfoLogger().Comment("NewJaegerTraceMgr: serviceName=%s", serviceName)

	t := &JaegerTraceMgr{
		level:       ulog.WarnLevel,
		mode:        "all",
		logHTTPBody: false,
		traceKeyMap: make(map[string]bool),
		fileWatcher: fileWatcher,
		configPath:  configPath,
	}

	if traceKeyArray != nil {
		for _, v := range traceKeyArray {
			t.traceKeyMap[v] = true
		}
	}

	t.defaultConf = jaegerConfig.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegerConfig.SamplerConfig{
			Type:                    jaeger.SamplerTypeConst,
			Param:                   1.0, // sample all traces
			SamplingRefreshInterval: 100 * time.Millisecond,
		},
		Reporter: &jaegerConfig.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 10 * time.Second,
			//LocalAgentHostPort:  "localhost:6832",
		},
	}

	t.ReloadConfig()

	if t.traceConf != nil && t.traceConf.ConfigFile() != "" {
		t.fileWatcher.Watch(t.traceConf.ConfigFile(), func() {
			t.ReloadConfig()
		})
	}

	return t
}

func (t *JaegerTraceMgr) useDefaultConf() {
	//loggers.InfoLogger().Comment("Use Default Config: ##############")
	tracer, closer, err := t.defaultConf.NewTracer(
		jaegerConfig.Logger(jaegerLogger),
	)

	if err == nil {
		//loggers.InfoLogger().Comment("ReloadConfig: ##############")
		if t.closer != nil {
			t.closer.Close()
		}
		t.tracer = tracer
		t.closer = closer
		//		opentracing.SetGlobalTracer(t.tracer)
		loggers.InfoLogger().Comment("[jaegertrace-onchange] %#v\n", t.defaultConf)
		loggers.InfoLogger().Comment("[jaegertrace-sampler-onchange] %#v\n", t.defaultConf.Sampler)
		loggers.InfoLogger().Comment("[jaegertrace-reporter-onchange] %#v\n", t.defaultConf.Reporter)
	}
}

func (t *JaegerTraceMgr) ReloadConfig() {
	defer func() {
		if r := recover(); r != nil {
			t.useDefaultConf()
		}
	}()

	//loggers.InfoLogger().Comment("ReloadConfig: configPath=%s", t.configPath)
	if t.configPath != "" {
		t.traceConf = testhelper.LoadConfigFromFile(t.configPath)
		var reloadConf jaegerConfig.Configuration
		var traceConf TraceConfig

		if err := t.traceConf.UnmarshalTo(&reloadConf); err == nil {
			tracer, closer, err := reloadConf.NewTracer(
				jaegerConfig.Logger(jaegerLogger),
			)
			if err == nil {
				if t.closer != nil {
					t.closer.Close()
				}
				t.tracer = tracer
				t.closer = closer
				//opentracing.SetGlobalTracer(t.tracer)
				loggers.InfoLogger().Comment("[jaegertrace-onchange] %#v\n", reloadConf)
				loggers.InfoLogger().Comment("[jaegertrace-sampler-onchange] %#v\n", reloadConf.Sampler)
				loggers.InfoLogger().Comment("[jaegertrace-reporter-onchange] %#v\n", reloadConf.Reporter)
			} else {
				loggers.ErrorLogger().Minor("ReloadConfig: err=%v", err)
			}
		}

		if err := t.traceConf.UnmarshalTo(&traceConf); err == nil {
			t.level = ulog.ParseLevel(traceConf.Level)
			t.mode = traceConf.Mode
			t.logHTTPBody = traceConf.LogHTTPBody

			loggers.InfoLogger().Comment("[utrace-onchange] %#v\n", traceConf)
		}

	} else {
		t.useDefaultConf()
	}
}

func (t *JaegerTraceMgr) Close() error {
	t.closer.Close()
	return nil
}

func (t *JaegerTraceMgr) Tracer() opentracing.Tracer {
	return t.tracer
}

func (t *JaegerTraceMgr) Level() ulog.LogLevel {
	return t.level
}

//func (t *JaegerTraceMgr) GinHTTPTraceHandler(serviceName string, level ulog.LogLevel, supi string) gin.HandlerFunc {
func (t *JaegerTraceMgr) GinHTTPTraceHandler(serviceName string, level ulog.LogLevel) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		start := time.Now()
		//loggers.InfoLogger().Comment("GinHTTPTraceHandler: before context=%p", ctx.Request.Context())
		span, req := t.StartSpanFromServerHTTPReq(ctx.Request, fmt.Sprintf("%s:/%s", serviceName, ctx.Request.URL.Path), level)
		ctx.Request = req
		if span != nil {
			defer span.Finish()
			//loggers.InfoLogger().Comment("GinHTTPTraceHandler: after context=%p", ctx.Request.Context())
		}

		// Process request
		ctx.Next()
		end := time.Now()

		if span != nil {
			latency := end.Sub(start)
			t.SetTags(level, ctx.Request, serviceName,
				"http.latency", latency,
				string(ext.HTTPStatusCode), uint16(ctx.Writer.Status()),
			)
		}
	}
}

func (t *JaegerTraceMgr) AtreugoHTTPTraceHandler(serviceName string, level ulog.LogLevel) atreugo.Middleware {
	return func(ctx *atreugo.RequestCtx) (int, error) {
		netHeader := http.Header{}
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			name := string(key)
			netHeader.Set(name, string(value))
		})

		span, traceContext := t.StartSpanFromHTTPHdr(netHeader, serviceName)
		if span != nil {
			defer span.Finish()
			ctx.SetUserValue(traceContextKeyStr, traceContext)
			t.SetTags(level, traceContext, serviceName,
				string(ext.SpanKind), string(ext.SpanKindRPCServerEnum),
				string(ext.HTTPUrl), string(ctx.RequestURI()),
				string(ext.HTTPMethod), string(ctx.Method()),
				"component", "github.com/savsgio/atreugo/v7",
			)

			if t.level <= level {
				t.LogFields(level, traceContext, serviceName,
					"request", ctx.Request.String(),
				)
			}
		}

		return http.StatusOK, nil
	}
}

func (t *JaegerTraceMgr) startSpanFromHTTPHdr(headers http.Header, serviceName string) (opentracing.Span, string) {
	var span opentracing.Span

	wireContext, err := t.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(headers))
	if err != nil {
		span = t.tracer.StartSpan(serviceName)
		//loggers.InfoLogger().Comment("startSpanFromHTTPHdr: Start root span. serviceName=%s, span=%v, err=%v, header=%v", serviceName, span, err, headers)
	} else {
		span = t.tracer.StartSpan(serviceName, opentracing.ChildOf(wireContext))
		//loggers.InfoLogger().Comment("startSpanFromHTTPHdr: Start span from http header. serviceName=%s, span=%v", serviceName, span)
	}

	traceKey := headers.Get(HeaderNames.TraceKey)

	return span, traceKey
}

func (t *JaegerTraceMgr) StartSpanFromServerHTTPReq(req *http.Request, serviceName string, level ulog.LogLevel) (opentracing.Span, *http.Request) {
	span, traceKey := t.startSpanFromHTTPHdr(req.Header, serviceName)
	if span == nil {
		loggers.ErrorLogger().Minor("StartSpanFromServerHTTPReq: Fail to start span. serviceName=%s", serviceName)
		return nil, nil
	}

	path := req.URL.Path
	raw := req.URL.RawQuery
	if len(raw) > 0 {
		path = path + "?" + raw
	}

	retReq := req
	if len(traceKey) > 0 {
		retReq = req.WithContext(t.SetTraceKey(req.Context(), traceKey))
		if retReq == nil {
			return nil, nil
		}
	} else {
		/* TO DO. json path 로 request 별 tracekey 추출 rule 설정 필요.
		traceKey := req.Header.Get(traceKeyHeaderName)
		*/
	}

	retReq = retReq.WithContext(opentracing.ContextWithSpan(retReq.Context(), span))

	t.SetTags(level, retReq, serviceName,
		string(ext.SpanKind), string(ext.SpanKindRPCServerEnum),
		string(ext.HTTPUrl), path,
		string(ext.HTTPMethod), retReq.Method,
		string(ext.PeerAddress), retReq.RemoteAddr,
		"component", "net/http",
	)

	if t.level <= level {
		msg, err := httputil.DumpRequest(retReq, t.logHTTPBody)
		if err != nil {
			t.LogFields(level, retReq, serviceName,
				"error", err.Error(),
			)
		} else {
			t.LogFields(level, retReq, serviceName,
				"request", string(msg),
			)
		}
	}

	return span, retReq
}

func (t *JaegerTraceMgr) StartSpanFromClientHTTPReq(req *http.Request, serviceName string, level ulog.LogLevel) (opentracing.Span, *http.Request) {
	span, traceKey := t.startSpanFromHTTPHdr(req.Header, serviceName)
	if span == nil {
		loggers.ErrorLogger().Minor("StartSpanFromClientHTTPReq: Fail to start span. serviceName=%s", serviceName)
		return nil, nil
	}

	//loggers.InfoLogger().Comment("N32cClient sapn = %v", span)

	path := req.URL.Path
	raw := req.URL.RawQuery
	if len(raw) > 0 {
		path = path + "?" + raw
	}

	retReq := req
	if len(traceKey) > 0 {
		retReq = req.WithContext(t.SetTraceKey(req.Context(), traceKey))
		if retReq == nil {
			return nil, nil
		}
	} else {
		/* TO DO. json path 로 request 별 tracekey 추출 rule 설정 필요.
		traceKey := req.Header.Get(traceKeyHeaderName)
		*/
	}

	retReq = retReq.WithContext(opentracing.ContextWithSpan(retReq.Context(), span))
	if retReq == nil {
		loggers.ErrorLogger().Minor("StartSpanFromClientHTTPReq: Fail to get req with context.")
		return nil, nil
	}

	t.InjectToHTTP(retReq.Context(), retReq.Header)

	t.SetTags(level, retReq, serviceName,
		string(ext.SpanKind), string(ext.SpanKindRPCClientEnum),
		string(ext.HTTPUrl), path,
		string(ext.HTTPMethod), retReq.Method,
		string(ext.PeerAddress), retReq.Host,
		"component", "net/http",
	)

	/*
		if t.level <= level {
			msg, err := httputil.DumpRequestOut(req, t.logHTTPBody)
			if err != nil {
				t.LogFields(level, retReq, serviceName,
					"error", err.Error(),
				)
			} else {
				t.LogFields(level, retReq, serviceName,
					"request", string(msg),
				)
			}
		}
	*/

	return span, retReq
}

func (t *JaegerTraceMgr) StartSpanFromHTTPHdr(headers http.Header, serviceName string) (opentracing.Span, context.Context) {
	span, traceKey := t.startSpanFromHTTPHdr(headers, serviceName)
	if span == nil {
		loggers.ErrorLogger().Minor("StartSpanFromHTTPHdr: Fail to start span. serviceName=%s", serviceName)
		return nil, nil
	}

	traceContext := context.Background()
	if len(traceKey) > 0 {
		traceContext = t.SetTraceKey(traceContext, traceKey)
	} else {
		/* TO DO. json path 로 request 별 tracekey 추출 rule 설정 필요.
		traceKey := req.Header.Get(traceKeyHeaderName)
		*/
	}
	traceContext = opentracing.ContextWithSpan(traceContext, span)

	return span, traceContext
}

func (t *JaegerTraceMgr) StartSpanFromServerHTTPHdr(headers http.Header, serviceName string, level ulog.LogLevel) (opentracing.Span, context.Context) {
	span, traceContext := t.StartSpanFromHTTPHdr(headers, serviceName)
	if span == nil {
		loggers.ErrorLogger().Minor("StartSpanFromHTTPHdr: Fail to start span. serviceName=%s", serviceName)
		return nil, nil
	}

	/*
		Something to log
	*/
	return span, traceContext
}

func (t *JaegerTraceMgr) StartSpanFromClientHTTPHdr(headers http.Header, serviceName string, level ulog.LogLevel) (opentracing.Span, context.Context) {
	span, traceContext := t.StartSpanFromHTTPHdr(headers, serviceName)
	if span == nil {
		loggers.ErrorLogger().Minor("StartSpanFromHTTPHdr: Fail to start span. serviceName=%s", serviceName)
		return nil, nil
	}

	/*
		Something to log
	*/
	return span, traceContext
}

func (t *JaegerTraceMgr) StartSpanFromContext(traceContext context.Context, serviceName string) (opentracing.Span, context.Context) {
	//loggers.InfoLogger().Comment("StartSpanFromContext: context=%p", traceContext)
	//loggers.InfoLogger().Comment("StartSpanFromContext: parentSpan=%p", opentracing.SpanFromContext(traceContext))
	return opentracing.StartSpanFromContextWithTracer(traceContext, t.tracer, serviceName)
}

func (t *JaegerTraceMgr) SpanFromHTTPReq(req *http.Request) opentracing.Span {
	return t.SpanFromContext(req.Context())
}

func (t *JaegerTraceMgr) SpanFromContext(traceContext context.Context) opentracing.Span {
	return opentracing.SpanFromContext(traceContext)
}

func (t *JaegerTraceMgr) InjectToHTTP(traceContext context.Context, headers http.Header) error {
	if traceContext == nil {
		return errcode.SystemError("Invalid context.")
	}

	span := opentracing.SpanFromContext(traceContext)
	if span == nil {
		return errcode.SystemError("Failed to create span from context.")
	}

	traceKey := t.GetTraceKey(traceContext)
	if len(traceKey) > 0 {
		headers.Set(HeaderNames.TraceKey, traceKey)
	}

	err := t.tracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(headers))
	//loggers.InfoLogger().Comment("InjectToHTTP: span=%v, headers=%v", span, headers)

	return err
}

func (t *JaegerTraceMgr) GetTraceKey(traceContext context.Context) string {
	val := traceContext.Value(traceContextKey)
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

func (t *JaegerTraceMgr) SetTraceKey(traceContext context.Context, traceKey string) context.Context {
	return context.WithValue(traceContext, traceContextKey, traceKey)
}

func (t *JaegerTraceMgr) logFields(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{}) {
	if t.level < level {
		//loggers.InfoLogger().Comment("level = %d < %d", t.level, level)
		return
	}

	if ctx == nil {
		loggers.ErrorLogger().Minor("Invalid Parameter. ctx is nil. serviceName=%s", serviceName)
		return
	}

	var span opentracing.Span
	var traceKey interface{}
	switch c := ctx.(type) {
	case context.Context:
		span = t.SpanFromContext(c)
		traceKey = t.GetTraceKey(c)
	case *http.Request:
		span = t.SpanFromHTTPReq(c)
		traceKey = t.GetTraceKey(c.Context())
	default:
		loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported ctx parameter type. serviceName=%s", serviceName)
		return
	}

	if span == nil {
		loggers.ErrorLogger().Minor("Fail to start span. serviceName=%s", serviceName)
		return
	}

	defaultFields := []interface{}{
		"event", "logs",
		"level", level.String(),
	}

	switch k := traceKey.(type) {
	case string:
		switch t.mode {
		case "all":
			f := append(defaultFields, keyValues...)
			span.LogKV(f...)
		case "subscriber":
			r, ok := t.traceKeyMap[k]
			if ok && r {
				f := append(defaultFields, keyValues...)
				span.LogKV(f...)
			}
		default:
			loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported trace mode=%s", t.mode)
		}
	default:
		loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported traceKey parameter type. serviceName=%s", serviceName)
	}
}

func (t *JaegerTraceMgr) LogFields(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(level, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsPanic(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.PanicLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsFatal(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.FatalLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsError(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.ErrorLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsWarn(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.WarnLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsInfo(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.InfoLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogFieldsDebug(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.logFields(ulog.DebugLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) setTags(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{}) {
	if t.level < level {
		//loggers.InfoLogger().Comment("level = %d < %d", t.level, level)
		return
	}

	if ctx == nil {
		loggers.ErrorLogger().Minor("Invalid Parameter. ctx is nil. serviceName=%s", serviceName)
		return
	}

	var span opentracing.Span
	var traceKey interface{}
	switch c := ctx.(type) {
	case context.Context:
		span = t.SpanFromContext(c)
		traceKey = t.GetTraceKey(c)
	case *http.Request:
		span = t.SpanFromHTTPReq(c)
		traceKey = t.GetTraceKey(c.Context())
	default:
		loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported ctx parameter type. serviceName=%s", serviceName)
		return
	}

	if span == nil {
		loggers.ErrorLogger().Minor("Fail to start span. serviceName=%s", serviceName)
		return
	}

	switch k := traceKey.(type) {
	case string:
		switch t.mode {
		case "all":
			err := KVToSetTag(span, keyValues...)
			if err != nil {
				loggers.ErrorLogger().Minor("err=%v", err)
			}
		case "subscriber":
			r, ok := t.traceKeyMap[k]
			if ok && r {
				err := KVToSetTag(span, keyValues...)
				if err != nil {
					loggers.ErrorLogger().Minor("err=%v", err)
				}
			}
		default:
			loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported trace mode=%s", t.mode)
		}
	default:
		loggers.ErrorLogger().Minor("Invalid Parameter. Unsupported traceKey parameter type. serviceName=%s", serviceName)
	}
}

func KVToSetTag(span opentracing.Span, keyValues ...interface{}) error {
	if len(keyValues)%2 != 0 {
		return fmt.Errorf("non-even keyValues len: %d, values=%v", len(keyValues), keyValues)
	}

	for i := 0; i*2 < len(keyValues); i++ {
		key, ok := keyValues[i*2].(string)
		if !ok {
			return fmt.Errorf(
				"non-string key (pair #%d): %T",
				i, keyValues[i*2])
		}
		val := keyValues[i*2+1]

		span.SetTag(key, val)
	}

	return nil
}

func (t *JaegerTraceMgr) SetTags(level ulog.LogLevel, ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(level, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) SetTagsPanic(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.PanicLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) SetTagsFatal(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.FatalLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) SetTagsError(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.ErrorLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) SetTagsWarn(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.WarnLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) SetTagsInfo(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.InfoLevel, ctx, serviceName, keyValues...)
}
func (t *JaegerTraceMgr) SetTagsDebug(ctx interface{}, serviceName string, keyValues ...interface{}) {
	t.setTags(ulog.DebugLevel, ctx, serviceName, keyValues...)
}

func (t *JaegerTraceMgr) LogHTTPRes(traceContext context.Context, res *http.Response, serviceName string, level ulog.LogLevel) {
	if traceContext == nil || res == nil {
		loggers.ErrorLogger().Minor("Invalid Parameter. traceContext=%p, res=%p", traceContext, res)
		return
	}

	t.SetTags(level, traceContext, serviceName,
		string(ext.HTTPStatusCode), uint16(res.StatusCode),
	)

	if t.level <= level {
		msg, err := httputil.DumpResponse(res, true)
		if err != nil {
			t.LogFields(level, traceContext, serviceName,
				"error", err.Error(),
			)
		} else {
			t.LogFields(level, traceContext, serviceName,
				"response", string(msg),
			)
		}
	}
}

func (t *JaegerTraceMgr) TraceHeaderName() string {
	/* TO DO */
	return HeaderNames.TraceID
}
