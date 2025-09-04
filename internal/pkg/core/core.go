package core

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"gin-example/configs"
	_ "gin-example/docs"
	"gin-example/internal/code"
	"gin-example/internal/pkg/color"
	"gin-example/internal/pkg/cors"
	"gin-example/internal/pkg/env"
	"gin-example/internal/pkg/errors"
	"gin-example/internal/pkg/timeutil"
	"gin-example/internal/pkg/trace"
	"gin-example/internal/proposal"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const _UI = `gin-example start success`

type Option func(*option)

type option struct {
	enablePProf      bool
	enableSwagger    bool
	enablePrometheus bool
	enableCors       bool
	alertNotify      proposal.AlertHandler
	recordHandler    proposal.RecordHandler
}

// WithEnablePProf 启用 pprof
func WithEnablePProf() Option {
	return func(opt *option) {
		opt.enablePProf = true
	}
}

// WithEnableSwagger 启用 swagger
func WithEnableSwagger() Option {
	return func(opt *option) {
		opt.enableSwagger = true
	}
}

// WithEnablePrometheus 启用 prometheus
func WithEnablePrometheus(recordHandler proposal.RecordHandler) Option {
	return func(opt *option) {
		opt.enablePrometheus = true
		opt.recordHandler = recordHandler
	}
}

// WithAlertNotify 设置告警通知
func WithAlertNotify(alertHandler proposal.AlertHandler) Option {
	return func(opt *option) {
		opt.alertNotify = alertHandler
	}
}

// WithEnableCors 设置支持跨域
func WithEnableCors() Option {
	return func(opt *option) {
		opt.enableCors = true
	}
}

// DisableTraceLog 禁止记录日志
func DisableTraceLog(ctx Context) {
	ctx.disableTrace()
}

// DisableRecordMetrics 禁止记录指标
func DisableRecordMetrics(ctx Context) {
	ctx.disableRecordMetrics()
}

// AliasForRecordMetrics 对请求路径起个别名，用于记录指标。
// 如：Get /user/:username 这样的路径，因为 username 会有非常多的情况，这样记录指标非常不友好。
func AliasForRecordMetrics(path string) HandlerFunc {
	return func(ctx Context) {
		ctx.setAlias(path)
	}
}

// WrapAuthHandler 用来处理 Auth 的入口
func WrapAuthHandler(handler func(Context) (sessionUserInfo proposal.SessionUserInfo, err BusinessError)) HandlerFunc {
	return func(ctx Context) {
		sessionUserInfo, err := handler(ctx)
		if err != nil {
			ctx.AbortWithError(err)
			return
		}

		ctx.setSessionUserInfo(sessionUserInfo)
	}
}

// RouterGroup 包装gin的RouterGroup
type RouterGroup interface {
	Group(string, ...HandlerFunc) RouterGroup
	IRoutes
}

var _ IRoutes = (*router)(nil)

// IRoutes 包装gin的IRoutes
type IRoutes interface {
	Any(string, ...HandlerFunc)
	GET(string, ...HandlerFunc)
	POST(string, ...HandlerFunc)
	DELETE(string, ...HandlerFunc)
	PATCH(string, ...HandlerFunc)
	PUT(string, ...HandlerFunc)
	OPTIONS(string, ...HandlerFunc)
	HEAD(string, ...HandlerFunc)
}

type router struct {
	group *gin.RouterGroup
}

func (r *router) Group(relativePath string, handlers ...HandlerFunc) RouterGroup {
	group := r.group.Group(relativePath, wrapHandlers(handlers...)...)
	return &router{group: group}
}

func (r *router) Any(relativePath string, handlers ...HandlerFunc) {
	r.group.Any(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) GET(relativePath string, handlers ...HandlerFunc) {
	r.group.GET(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) POST(relativePath string, handlers ...HandlerFunc) {
	r.group.POST(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) DELETE(relativePath string, handlers ...HandlerFunc) {
	r.group.DELETE(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) PATCH(relativePath string, handlers ...HandlerFunc) {
	r.group.PATCH(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) PUT(relativePath string, handlers ...HandlerFunc) {
	r.group.PUT(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) OPTIONS(relativePath string, handlers ...HandlerFunc) {
	r.group.OPTIONS(relativePath, wrapHandlers(handlers...)...)
}

func (r *router) HEAD(relativePath string, handlers ...HandlerFunc) {
	r.group.HEAD(relativePath, wrapHandlers(handlers...)...)
}

func wrapHandlers(handlers ...HandlerFunc) []gin.HandlerFunc {
	funcs := make([]gin.HandlerFunc, len(handlers))
	for i, handler := range handlers {
		handler := handler
		funcs[i] = func(c *gin.Context) {
			ctx := newContext(c)
			defer releaseContext(ctx)

			handler(ctx)
		}
	}

	return funcs
}

var _ Mux = (*mux)(nil)

// Mux http mux
type Mux interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request)
	Group(relativePath string, handlers ...HandlerFunc) RouterGroup
	Routes() gin.RoutesInfo
}

type mux struct {
	engine *gin.Engine
}

func (m *mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.engine.ServeHTTP(w, req)
}

func (m *mux) Group(relativePath string, handlers ...HandlerFunc) RouterGroup {
	return &router{
		group: m.engine.Group(relativePath, wrapHandlers(handlers...)...),
	}
}

func (m *mux) Routes() gin.RoutesInfo {
	return m.engine.Routes()
}

func New(logger *zap.Logger, options ...Option) (Mux, error) {
	if logger == nil {
		return nil, errors.New("logger required")
	}

	gin.SetMode(gin.ReleaseMode)
	mux := &mux{
		engine: gin.New(),
	}

	fmt.Println(color.Blue(_UI))

	// withoutTracePaths 这些请求，默认不记录日志
	withoutTracePaths := map[string]bool{
		"/metrics": true,

		"/debug/pprof/":             true,
		"/debug/pprof/cmdline":      true,
		"/debug/pprof/profile":      true,
		"/debug/pprof/symbol":       true,
		"/debug/pprof/trace":        true,
		"/debug/pprof/allocs":       true,
		"/debug/pprof/block":        true,
		"/debug/pprof/goroutine":    true,
		"/debug/pprof/heap":         true,
		"/debug/pprof/mutex":        true,
		"/debug/pprof/threadcreate": true,

		"/favicon.ico": true,

		"/system/health": true,
	}

	opt := new(option)
	for _, f := range options {
		f(opt)
	}

	if opt.enablePProf {
		if !env.Active().IsPro() {
			pprof.Register(mux.engine) // register pprof to gin
		}
	}

	if opt.enableSwagger {
		if !env.Active().IsPro() {
			mux.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // register swagger
		}
	}

	if opt.enablePrometheus {
		mux.engine.GET("/metrics", gin.WrapH(promhttp.Handler())) // register prometheus
	}

	if opt.enableCors {
		mux.engine.Use(cors.New())
	}

	// recover 两次，防止 recover 过程中时发生 panic
	mux.engine.Use(func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("got panic", zap.String("panic", fmt.Sprintf("%+v", err)), zap.String("stack", string(debug.Stack())))
			}
		}()

		ctx.Next()
	})

	mux.engine.Use(func(ctx *gin.Context) {

		if ctx.Writer.Status() == http.StatusNotFound {
			return
		}

		ts := time.Now()

		context := newContext(ctx)
		defer releaseContext(context)

		context.init()
		context.setLogger(logger)
		context.ableRecordMetrics()

		if !withoutTracePaths[ctx.Request.URL.Path] {
			if traceId := context.GetHeader(trace.Header); traceId != "" {
				context.setTrace(trace.New(traceId))
			} else {
				context.setTrace(trace.New(""))
			}
		}

		defer func() {
			var (
				response        interface{}
				businessCode    int
				businessCodeMsg string
				abortErr        error
				traceId         string
			)

			if ct := context.Trace(); ct != nil {
				context.SetHeader(trace.Header, ct.ID())
				traceId = ct.ID()
			}

			// region 发生 Panic 异常发送告警提醒
			if err := recover(); err != nil {
				stackInfo := string(debug.Stack())
				logger.Error("got panic", zap.String("panic", fmt.Sprintf("%+v", err)), zap.String("stack", stackInfo))
				context.AbortWithError(Error(
					http.StatusInternalServerError,
					code.ServerError,
					code.Text(code.ServerError)),
				)

				if alertHandler := opt.alertNotify; alertHandler != nil {
					alertHandler(&proposal.AlertMessage{
						ProjectName:  configs.ProjectName,
						Env:          env.Active().Value(),
						TraceID:      traceId,
						HOST:         context.Host(),
						URI:          context.URI(),
						Method:       context.Method(),
						ErrorMessage: err,
						ErrorStack:   stackInfo,
						Time:         time.Now().Format(timeutil.CSTLayout),
					})
				}
			}
			// endregion

			// region 发生错误，进行返回
			if ctx.IsAborted() {
				for i := range ctx.Errors {
					multierr.AppendInto(&abortErr, ctx.Errors[i])
				}

				if err := context.abortError(); err != nil { // customer err
					// 判断是否需要发送告警通知
					if err.IsAlert() {
						if alertHandler := opt.alertNotify; alertHandler != nil {
							alertHandler(&proposal.AlertMessage{
								ProjectName:  configs.ProjectName,
								Env:          env.Active().Value(),
								TraceID:      traceId,
								HOST:         context.Host(),
								URI:          context.URI(),
								Method:       context.Method(),
								ErrorMessage: err.Message(),
								ErrorStack:   fmt.Sprintf("%+v", err.StackError()),
								Time:         time.Now().Format(timeutil.CSTLayout),
							})
						}
					}

					multierr.AppendInto(&abortErr, err.StackError())
					businessCode = err.BusinessCode()
					businessCodeMsg = err.Message()
					response = &code.Failure{
						Code:    businessCode,
						Message: businessCodeMsg,
					}
					ctx.JSON(err.HTTPCode(), response)
				}
			}
			// endregion

			// region 正确返回
			response = context.getPayload()
			if response != nil {
				ctx.JSON(http.StatusOK, response)
			}
			// endregion

			// region 记录指标
			if opt.recordHandler != nil && context.isRecordMetrics() {
				path := context.Path()
				if alias := context.Alias(); alias != "" {
					path = alias
				}

				opt.recordHandler(&proposal.MetricsMessage{
					HOST:         context.Host(),
					Path:         path,
					Method:       context.Method(),
					HTTPCode:     ctx.Writer.Status(),
					BusinessCode: businessCode,
					CostSeconds:  time.Since(ts).Seconds(),
					IsSuccess:    !ctx.IsAborted() && (ctx.Writer.Status() == http.StatusOK),
				})
			}
			// endregion

			// region 记录日志
			var t *trace.Trace
			if x := context.Trace(); x != nil {
				t = x.(*trace.Trace)
			} else {
				return
			}

			decodedURL, _ := url.QueryUnescape(ctx.Request.URL.RequestURI())

			// ctx.Request.Header，精简 Header 参数
			traceHeader := map[string]string{
				"Content-Type": ctx.GetHeader("Content-Type"),
			}

			t.WithRequest(&trace.Request{
				TTL:        "un-limit",
				Method:     ctx.Request.Method,
				DecodedURL: decodedURL,
				Header:     traceHeader,
				Body:       string(context.RawData()),
			})

			var responseBody interface{}

			if response != nil {
				responseBody = response
			}

			t.WithResponse(&trace.Response{
				Header:          ctx.Writer.Header(),
				HttpCode:        ctx.Writer.Status(),
				HttpCodeMsg:     http.StatusText(ctx.Writer.Status()),
				BusinessCode:    businessCode,
				BusinessCodeMsg: businessCodeMsg,
				Body:            responseBody,
				CostSeconds:     time.Since(ts).Seconds(),
			})

			t.Success = !ctx.IsAborted() && (ctx.Writer.Status() == http.StatusOK)
			t.CostSeconds = time.Since(ts).Seconds()

			logger.Info("trace-log",
				zap.Any("method", ctx.Request.Method),
				zap.Any("path", decodedURL),
				zap.Any("http_code", ctx.Writer.Status()),
				zap.Any("business_code", businessCode),
				zap.Any("success", t.Success),
				zap.Any("cost_seconds", t.CostSeconds),
				zap.Any("trace_id", t.Identifier),
				zap.Any("trace_info", t),
				zap.Error(abortErr),
			)
			// endregion
		}()

		ctx.Next()
	})

	mux.engine.NoMethod(wrapHandlers(DisableTraceLog)...)
	mux.engine.NoRoute(wrapHandlers(DisableTraceLog)...)

	return mux, nil
}
