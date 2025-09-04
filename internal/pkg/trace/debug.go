package trace

type Debug struct {
	Stack string `json:"stack"` // 文件地址和行号
	Value []any  `json:"value"` // 值
}
