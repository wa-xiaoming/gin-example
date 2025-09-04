package shutdown

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	hooks   []func()
	hooksMu sync.Mutex
)

// RegisterHook 注册关闭钩子函数
func RegisterHook(hook func()) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	
	hooks = append(hooks, hook)
}

// Close 监听signal并停止
func Close(handler func()) {
	ctx := make(chan os.Signal, 1)
	signal.Notify(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	<-ctx
	signal.Stop(ctx)

	// 执行所有注册的钩子函数
	hooksMu.Lock()
	for _, hook := range hooks {
		hook()
	}
	hooksMu.Unlock()

	handler()
}