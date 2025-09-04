package proposal

import (
	"encoding/json"
)

// AlertMessage 告警信息
type AlertMessage struct {
	ProjectName  string      `json:"project_name"`  // 项目名称
	Env          string      `json:"env"`           // 运行环境
	TraceID      string      `json:"trace_id"`      // 当前请求的唯一ID
	HOST         string      `json:"host"`          // 当前请求的 HOST
	URI          string      `json:"uri"`           // 当前请求的 URI
	Method       string      `json:"method"`        // 当前请求的 Method
	ErrorMessage interface{} `json:"error_message"` // 错误信息
	ErrorStack   string      `json:"error_stack"`   // 堆栈信息
	Time         string      `json:"time"`          // 发生时间
}

// Marshal 序列化到JSON
func (a *AlertMessage) Marshal() (jsonRaw []byte) {
	jsonRaw, _ = json.Marshal(a)
	return
}

// AlertHandler 告警处理的句柄
type AlertHandler func(msg *AlertMessage)
