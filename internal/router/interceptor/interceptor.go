package interceptor

import (
	"time"

	"gin-example/internal/pkg/circuitbreaker"
	"gin-example/internal/pkg/ratelimit"

	"go.uber.org/zap"
)

// Interceptor 拦截器
type Interceptor struct {
	logger         *zap.Logger
	rateLimiter    ratelimit.Limiter
	circuitBreaker *circuitbreaker.CircuitBreaker
}

// NewInterceptor 创建拦截器
func NewInterceptor(logger *zap.Logger) *Interceptor {
	// 创建令牌桶限流器，每秒100个请求
	rateLimiter := ratelimit.NewTokenBucketLimiter(100)
	
	// 创建熔断器配置
	config := &circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          time.Second,
		ResetTimeout:     time.Second * 10,
		SuccessThreshold: 1,
		ServiceName:      "http-server",
	}
	
	// 创建熔断器
	circuitBreaker := circuitbreaker.NewCircuitBreaker(config)

	return &Interceptor{
		logger:         logger,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
	}
}

// GetRateLimiter 获取限流器
func (i *Interceptor) GetRateLimiter() ratelimit.Limiter {
	return i.rateLimiter
}

// GetCircuitBreaker 获取熔断器
func (i *Interceptor) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return i.circuitBreaker
}