package trace

type Mongo struct {
	Time        string  `json:"time"`         // 时间，格式：2006-01-02 15:04:05
	Database    string  `json:"database"`     // 数据库名称
	Command     string  `json:"command"`      // 命令
	Reply       string  `json:"reply"`        // 响应
	CostSeconds float64 `json:"cost_seconds"` // 执行时间(单位秒)
}
