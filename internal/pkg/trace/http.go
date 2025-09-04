package trace

type HttpLog struct {
	Request     map[string]interface{} `json:"request"`      // 请求信息
	Response    map[string]interface{} `json:"response"`     // 响应信息
	CostSeconds float64                `json:"cost_seconds"` // 执行时间(单位秒)
}
