package circuitbreaker

import (
	"errors"
	"sync"
	"time"

	"gin-example/internal/metrics"
)

// State 熔断器状态
type State int

const (
	// Closed 关闭状态
	Closed State = iota
	// Open 开启状态
	Open
	// HalfOpen 半开启状态
	HalfOpen
)

// String 实现State的字符串表示
func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	// 配置参数
	failureThreshold int           // 失败次数阈值
	timeout          time.Duration // 超时时间
	resetTimeout     time.Duration // 重置超时时间
	successThreshold int           // 半开启状态下成功阈值

	// 状态
	state State // 当前状态

	// 统计信息
	failureCount   int       // 连续失败次数
	lastFailure    time.Time // 上次失败时间
	trippedTime    time.Time // 熔断触发时间
	successCount   int       // 半开启状态下成功次数
	requestCount   int       // 总请求数
	totalSuccess   int       // 总成功数
	totalFailure   int       // 总失败数

	// 服务标识
	serviceName    string    // 服务名称，用于监控指标

	// 锁
	mutex sync.Mutex
}

// Config 熔断器配置
type Config struct {
	FailureThreshold int           // 失败次数阈值
	Timeout          time.Duration // 超时时间
	ResetTimeout     time.Duration // 重置超时时间
	SuccessThreshold int           // 半开启状态下成功阈值
	ServiceName      string        // 服务名称
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold: 5,
		Timeout:          time.Second * 60,
		ResetTimeout:     time.Second * 30,
		SuccessThreshold: 1,
		ServiceName:      "unknown",
	}
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	cb := &CircuitBreaker{
		failureThreshold: config.FailureThreshold,
		timeout:          config.Timeout,
		resetTimeout:     config.ResetTimeout,
		successThreshold: config.SuccessThreshold,
		state:            Closed,
		serviceName:      config.ServiceName,
	}

	// 初始化监控指标
	metrics.SetCircuitBreakerState(cb.serviceName, float64(cb.state))

	return cb
}

// Execute 执行受保护的函数
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 检查是否可以执行请求
	if !cb.canExecute() {
		// 记录熔断器触发事件
		metrics.RecordCircuitBreakerTripped(cb.serviceName)
		return errors.New("circuit breaker is open")
	}

	// 增加请求数
	cb.requestCount++

	// 执行函数
	err := fn()

	// 更新状态
	cb.updateState(err)

	// 更新监控指标
	metrics.SetCircuitBreakerState(cb.serviceName, float64(cb.state))

	return err
}

// canExecute 是否可以执行请求
func (cb *CircuitBreaker) canExecute() bool {
	switch cb.state {
	case Closed:
		return true
	case Open:
		// 检查是否可以进入半开启状态
		if time.Since(cb.trippedTime) >= cb.resetTimeout {
			cb.state = HalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case HalfOpen:
		// 半开启状态下只允许有限的请求通过
		return cb.successCount < cb.successThreshold
	default:
		return false
	}
}

// updateState 更新状态
func (cb *CircuitBreaker) updateState(err error) {
	cb.requestCount++

	if err != nil {
		// 请求失败
		cb.failureCount++
		cb.totalFailure++
		cb.lastFailure = time.Now()

		switch cb.state {
		case Closed:
			// 检查是否需要打开熔断器
			if cb.failureCount >= cb.failureThreshold {
				cb.state = Open
				cb.trippedTime = time.Now()
			}
		case HalfOpen:
			// 半开启状态下失败，重新打开熔断器
			cb.state = Open
			cb.trippedTime = time.Now()
		}
	} else {
		// 请求成功
		cb.totalSuccess++

		switch cb.state {
		case Closed:
			// 重置失败计数
			cb.failureCount = 0
		case HalfOpen:
			// 半开启状态下成功，增加成功计数
			cb.successCount++
			// 如果连续成功次数达到阈值，则关闭熔断器
			if cb.successCount >= cb.successThreshold {
				cb.state = Closed
				cb.failureCount = 0
			}
		}
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}

// Metrics 获取统计信息
func (cb *CircuitBreaker) Metrics() map[string]interface{} {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return map[string]interface{}{
		"state":            cb.state.String(),
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"request_count":    cb.requestCount,
		"total_success":    cb.totalSuccess,
		"total_failure":    cb.totalFailure,
		"last_failure":     cb.lastFailure,
		"tripped_time":     cb.trippedTime,
		"service_name":     cb.serviceName,
		"failure_threshold": cb.failureThreshold,
		"success_threshold": cb.successThreshold,
		"timeout":          cb.timeout,
		"reset_timeout":    cb.resetTimeout,
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailure = time.Time{}
	cb.trippedTime = time.Time{}
	
	// 更新监控指标
	metrics.SetCircuitBreakerState(cb.serviceName, float64(cb.state))
}

// IsOpen 熔断器是否开启
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state == Open
}

// IsClosed 熔断器是否关闭
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state == Closed
}

// IsHalfOpen 熔断器是否半开启
func (cb *CircuitBreaker) IsHalfOpen() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state == HalfOpen
}