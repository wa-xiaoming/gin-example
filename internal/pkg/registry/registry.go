package registry

// Registry 服务注册接口
type Registry interface {
	// Register 注册服务
	Register() error
	
	// Deregister 注销服务
	Deregister() error
}