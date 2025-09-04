package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"gin-example/internal/code"
	"gin-example/internal/metrics"
	"gin-example/internal/pkg/core"

	"go.uber.org/ratelimit"
	"golang.org/x/time/rate"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 全局限流配置
	GlobalRPS    int  // 每秒请求数
	GlobalBurst  int  // 突发请求数
	EnableGlobal bool // 是否启用全局限流

	// IP限流配置
	IPLimitRPS    int  // 每个IP每秒请求数
	IPLimitBurst  int  // 每个IP突发请求数
	EnableIPLimit bool // 是否启用IP限流

	// 用户限流配置
	UserLimitRPS    int  // 每个用户每秒请求数
	UserLimitBurst  int  // 每个用户突发请求数
	EnableUserLimit bool // 是否启用用户限流

	// 限流响应配置
	ErrorMessage string // 限流时的错误消息
	ErrorCode    int    // 限流时的错误码
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalRPS:       1000,
		GlobalBurst:     100,
		EnableGlobal:    true,
		IPLimitRPS:      100,
		IPLimitBurst:    10,
		EnableIPLimit:   true,
		UserLimitRPS:    50,
		UserLimitBurst:  5,
		EnableUserLimit: false,
		ErrorMessage:    "请求过于频繁，请稍后再试",
		ErrorCode:       http.StatusTooManyRequests,
	}
}

// RateLimitMiddleware 增强的限流中间件
type RateLimitMiddleware struct {
	config        *RateLimitConfig
	globalLimiter rate.Limiter
	ipLimiters    map[string]*rate.Limiter
	userLimiters  map[string]*rate.Limiter
	limiterPool   sync.Pool
	mu            sync.RWMutex
}

// NewRateLimitMiddleware 创建新的限流中间件
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimitMiddleware{
		config:       config,
		ipLimiters:   make(map[string]*rate.Limiter),
		userLimiters: make(map[string]*rate.Limiter),
	}

	// 初始化全局限流器
	if config.EnableGlobal {
		rl.globalLimiter = *rate.NewLimiter(rate.Limit(config.GlobalRPS), config.GlobalBurst)
	}

	// 初始化对象池
	rl.limiterPool = sync.Pool{
		New: func() interface{} {
			return ratelimit.New(100) // 默认令牌桶限流器
		},
	}

	// 启动定期清理过期限流器的goroutine
	go rl.cleanupExpiredLimiters()

	return rl
}

// Middleware 限流中间件函数
func (rl *RateLimitMiddleware) Middleware() core.HandlerFunc {
	return func(ctx core.Context) {
		// 全局限流检查
		if rl.config.EnableGlobal && !rl.globalLimiter.Allow() {
			rl.handleRateLimitExceeded(ctx, "global")
			return
		}

		// IP限流检查
		if rl.config.EnableIPLimit {
			ip := rl.getClientIP(ctx)
			if !rl.getIPLimiter(ip).Allow() {
				rl.handleRateLimitExceeded(ctx, "ip:"+ip)
				return
			}
		}

		// 用户限流检查
		if rl.config.EnableUserLimit {
			userID := rl.getUserID(ctx)
			if !rl.getUserLimiter(userID).Allow() {
				rl.handleRateLimitExceeded(ctx, "user:"+userID)
				return
			}
		}

		// 记录限流指标
		metrics.RecordMetrics("RATE_LIMIT", "ALLOWED", true, 200, 0, 0)

		// 继续执行下一个处理器
		ctx.Next()
	}
}

// handleRateLimitExceeded 处理限流超限情况
func (rl *RateLimitMiddleware) handleRateLimitExceeded(ctx core.Context, limitType string) {
	// 记录限流指标
	metrics.RecordMetrics("RATE_LIMIT", "EXCEEDED", false, rl.config.ErrorCode, code.ServerError, 0)
	metrics.RecordError("rate_limit", limitType)

	ctx.AbortWithError(core.Error(
		rl.config.ErrorCode,
		code.ServerError,
		rl.config.ErrorMessage,
	))
}

// getClientIP 获取客户端IP地址
func (rl *RateLimitMiddleware) getClientIP(ctx core.Context) string {
	// 尝试从标准头部获取
	ip := ctx.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = ctx.GetHeader("X-Real-IP")
	}

	// 如果头部中没有IP信息，则使用远程地址
	if ip == "" {
		ip = rl.extractIP(ctx.RemoteAddr())
	}

	return ip
}

// getUserID 获取用户ID
func (rl *RateLimitMiddleware) getUserID(ctx core.Context) string {
	// 从请求头或上下文中获取用户ID
	userID := ctx.GetHeader("User-ID")
	if userID == "" {
		// 示例：可以从JWT token中提取用户ID
		// 这里简化处理，返回anonymous
		userID = "anonymous"
	}

	return userID
}

// extractIP 从远程地址中提取IP
func (rl *RateLimitMiddleware) extractIP(addr string) string {
	if addr == "" {
		return ""
	}

	// 获取RemoteAddr并去除端口号
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// 如果没有端口号，直接返回addr
		host = addr
	}

	return host
}

// getIPLimiter 获取IP对应的限流器
func (rl *RateLimitMiddleware) getIPLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// 双重检查
		limiter, exists = rl.ipLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rl.config.IPLimitRPS), rl.config.IPLimitBurst)
			rl.ipLimiters[ip] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

// getUserLimiter 获取用户对应的限流器
func (rl *RateLimitMiddleware) getUserLimiter(userID string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.userLimiters[userID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// 双重检查
		limiter, exists = rl.userLimiters[userID]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rl.config.UserLimitRPS), rl.config.UserLimitBurst)
			rl.userLimiters[userID] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

// cleanupExpiredLimiters 定期清理过期的限流器
func (rl *RateLimitMiddleware) cleanupExpiredLimiters() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupLimiters()
		}
	}
}

// cleanupLimiters 清理限流器
func (rl *RateLimitMiddleware) cleanupLimiters() {
	// 简化实现：实际项目中可以根据最后使用时间来清理
	// 这里仅记录日志
	rl.mu.RLock()
	ipCount := len(rl.ipLimiters)
	userCount := len(rl.userLimiters)
	rl.mu.RUnlock()

	// 记录指标
	// 注意：这里只是示例，实际项目中应该根据需要记录
	_ = ipCount
	_ = userCount
}
