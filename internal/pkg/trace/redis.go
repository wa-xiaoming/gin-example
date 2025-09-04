package trace

type Redis struct {
	Time        string      `json:"time"`         // 时间，格式：2006-01-02 15:04:05
	Stack       string      `json:"stack"`        // 文件地址和行号
	Cmd         interface{} `json:"cmd"`          // 操作
	CostSeconds float64     `json:"cost_seconds"` // 执行时间(单位秒)
}
