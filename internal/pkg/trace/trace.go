package trace

import (
	"sync"

	"gin-example/internal/pkg/idgen"
)

const Header = "TRACE-ID"

var _ T = (*Trace)(nil)

type T interface {
	i()
	ID() string
	WithRequest(req *Request) *Trace
	WithResponse(resp *Response) *Trace
	AppendThirdPartyRequests(http *HttpLog) *Trace
	AppendSQL(sql *SQL) *Trace
	AppendDebug(debug *Debug) *Trace
	AppendRedis(redis *Redis) *Trace
	AppendMongo(mongo *Mongo) *Trace
}

// Trace 记录的参数
type Trace struct {
	mux                sync.Mutex
	Identifier         string     `json:"trace_id"`             // 链路ID
	Request            *Request   `json:"request"`              // 请求信息
	Response           *Response  `json:"response"`             // 返回信息
	ThirdPartyRequests []*HttpLog `json:"third_party_requests"` // 调用第三方接口的信息
	Debugs             []*Debug   `json:"debugs"`               // 调试信息
	SQLs               []*SQL     `json:"sqls"`                 // 执行的 SQL 信息
	Redis              []*Redis   `json:"redis"`                // 执行的 Redis 信息
	Mongos             []*Mongo   `json:"mongos"`               // 执行的 Mongo 信息
	Success            bool       `json:"success"`              // 请求结果 true or false
	CostSeconds        float64    `json:"cost_seconds"`         // 执行时长(单位秒)
}

// Request 请求信息
type Request struct {
	TTL        string      `json:"ttl"`         // 请求超时时间
	Method     string      `json:"method"`      // 请求方式
	DecodedURL string      `json:"decoded_url"` // 请求地址
	Header     interface{} `json:"header"`      // 请求 Header 信息
	Body       interface{} `json:"body"`        // 请求 Body 信息
}

// Response 响应信息
type Response struct {
	Header          interface{} `json:"header"`                      // Header 信息
	Body            interface{} `json:"body"`                        // Body 信息
	BusinessCode    int         `json:"business_code,omitempty"`     // 业务码
	BusinessCodeMsg string      `json:"business_code_msg,omitempty"` // 提示信息
	HttpCode        int         `json:"http_code"`                   // HTTP 状态码
	HttpCodeMsg     string      `json:"http_code_msg"`               // HTTP 状态码信息
	CostSeconds     float64     `json:"cost_seconds"`                // 执行时间(单位秒)
}

func New(id string) *Trace {
	if id == "" {
		id = idgen.GenerateUniqueID()
	}

	return &Trace{
		Identifier: id,
	}
}

func (t *Trace) i() {}

// ID 唯一标识符
func (t *Trace) ID() string {
	return t.Identifier
}

// WithRequest 设置 request 信息
func (t *Trace) WithRequest(req *Request) *Trace {
	t.Request = req
	return t
}

// WithResponse 设置 response 信息
func (t *Trace) WithResponse(resp *Response) *Trace {
	t.Response = resp
	return t
}

// AppendThirdPartyRequests 追加 HTTP 三方请求日志
func (t *Trace) AppendThirdPartyRequests(http *HttpLog) *Trace {
	if http == nil {
		return t
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	t.ThirdPartyRequests = append(t.ThirdPartyRequests, http)
	return t
}

// AppendDebug 追加 debug 日志
func (t *Trace) AppendDebug(debug *Debug) *Trace {
	if debug == nil {
		return t
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	t.Debugs = append(t.Debugs, debug)
	return t
}

// AppendSQL 追加 SQL 执行日志
func (t *Trace) AppendSQL(sql *SQL) *Trace {
	if sql == nil {
		return t
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	t.SQLs = append(t.SQLs, sql)
	return t
}

// AppendRedis 追加 Redis 执行日志
func (t *Trace) AppendRedis(redis *Redis) *Trace {
	if redis == nil {
		return t
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	t.Redis = append(t.Redis, redis)
	return t
}

// AppendMongo 追加 Mongo 执行日志
func (t *Trace) AppendMongo(mongo *Mongo) *Trace {
	if mongo == nil {
		return t
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	t.Mongos = append(t.Mongos, mongo)
	return t
}
