package ratelimit

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"gin-example/internal/code"
	"gin-example/internal/pkg/core"

	"go.uber.org/ratelimit"
	"golang.org/x/time/rate"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 是否允许通过
	Allow() bool

	// AllowWithKey 基于键值的限流
	AllowWithKey(key string) bool
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	limiter ratelimit.Limiter
}

// NewTokenBucketLimiter 创建令牌桶限流器
// rate 每秒生成令牌数
func NewTokenBucketLimiter(rate int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		limiter: ratelimit.New(rate),
	}
}

// Allow 是否允许通过
func (l *TokenBucketLimiter) Allow() bool {
	// 消费一个令牌，如果没可用令牌会阻塞
	l.limiter.Take()
	return true
}

// AllowWithKey 基于键值的限流（令牌桶不支持键值限流，直接返回true）
func (l *TokenBucketLimiter) AllowWithKey(key string) bool {
	return l.Allow()
}

// LeakyBucketLimiter 漏桶限流器
type LeakyBucketLimiter struct {
	rate    rate.Limit // 令牌生成速率（每秒生成的令牌数）
	burst   int        // 桶容量
	mu      sync.Mutex
	buckets map[string]*rate.Limiter // 每个键对应一个限流器
}

// NewLeakyBucketLimiter 创建漏桶限流器
func NewLeakyBucketLimiter(ratePerSecond float64, burst int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		rate:    rate.Limit(ratePerSecond),
		burst:   burst,
		buckets: make(map[string]*rate.Limiter),
	}
}

// Allow 是否允许通过（使用默认键）
func (l *LeakyBucketLimiter) Allow() bool {
	return l.AllowWithKey("default")
}

// AllowWithKey 基于键值的限流
func (l *LeakyBucketLimiter) AllowWithKey(key string) bool {
	l.mu.Lock()
	limiter, exists := l.buckets[key]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.buckets[key] = limiter
	}
	l.mu.Unlock()

	return limiter.Allow()
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	limit   int           // 限制请求数
	window  time.Duration // 时间窗口
	mu      sync.Mutex
	records map[string][]time.Time // 每个键对应的请求时间记录
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:   limit,
		window:  window,
		records: make(map[string][]time.Time),
	}
}

// Allow 是否允许通过（使用默认键）
func (s *SlidingWindowLimiter) Allow() bool {
	return s.AllowWithKey("default")
}

// AllowWithKey 基于键值的限流
func (s *SlidingWindowLimiter) AllowWithKey(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// 清理过期记录
	cutoff := now.Add(-s.window)
	validRecords := make([]time.Time, 0)
	for _, t := range s.records[key] {
		if t.After(cutoff) {
			validRecords = append(validRecords, t)
		}
	}

	// 检查是否超过限制
	if len(validRecords) >= s.limit {
		return false
	}

	// 添加当前请求记录
	s.records[key] = append(validRecords, now)
	return true
}

// RateLimitMiddlewareWithIP 限流中间件（基于IP限流）
func RateLimitMiddlewareWithIP(limiter Limiter) core.HandlerFunc {
	return func(ctx core.Context) {
		// 从核心上下文中获取客户端IP
		ip := ctx.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = ctx.GetHeader("X-Real-IP")
		}
		// 如果头部中没有IP信息，则使用远程地址
		if ip == "" {
			// 从请求中提取IP地址
			ip = extractIP(ctx.RemoteAddr())
		}

		if !limiter.AllowWithKey(ip) {
			ctx.AbortWithError(core.Error(
				http.StatusTooManyRequests,
				code.ServerError,
				"请求过于频繁，请稍后再试",
			))
			return
		}

		// 继续执行下一个处理器
		ctx.Next()
	}
}

// RateLimitMiddlewareWithUser 限流中间件（基于用户ID限流）
func RateLimitMiddlewareWithUser(limiter Limiter) core.HandlerFunc {
	return func(ctx core.Context) {
		// 获取用户ID（示例中使用固定值，实际项目中应从上下文中获取）
		userID := "anonymous"
		if u := ctx.GetHeader("User-ID"); u != "" {
			userID = u
		}

		if !limiter.AllowWithKey(userID) {
			ctx.AbortWithError(core.Error(
				http.StatusTooManyRequests,
				code.ServerError,
				"请求过于频繁，请稍后再试",
			))
			return
		}

		// 继续执行下一个处理器
		ctx.Next()
	}
}

// extractIP 从请求中提取客户端IP地址
func extractIP(addr string) string {
	if addr == "" {
		return ""
	}

	// 获取RemoteAddr并去除端口号
	ip := addr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}

	// 处理IPv6地址
	if strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		ip = ip[1 : len(ip)-1]
	}

	return ip
}
