package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// Config 重试配置
type Config struct {
	MaxRetries  int           // 最大重试次数
	MinInterval time.Duration // 最小重试间隔
	MaxInterval time.Duration // 最大重试间隔
	MaxJitter   time.Duration // 最大抖动时间
	Timeout     time.Duration // 操作超时时间
}

// DefaultConfig 默认重试配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:  3,
		MinInterval: 100 * time.Millisecond,
		MaxInterval: 1 * time.Second,
		MaxJitter:   100 * time.Millisecond,
		Timeout:     5 * time.Second,
	}
}

// RetryableFunc 可重试的函数
type RetryableFunc func() error

// RetryWithConfig 带配置的重试函数
func RetryWithConfig(ctx context.Context, logger *zap.Logger, config *Config, fn RetryableFunc) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	
	for i := 0; i <= config.MaxRetries; i++ {
		// 创建带超时的上下文并立即执行函数
		func() {
			opCtx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel() // 确保及时释放资源
			
			// 在超时上下文中执行函数
			lastErr = fn()
			
			// 检查上下文是否超时或被取消
			select {
			case <-opCtx.Done():
				if lastErr == nil {
					lastErr = opCtx.Err()
				}
			default:
			}
		}()
		
		// 如果成功执行，直接返回
		if lastErr == nil {
			// 成功执行
			if i > 0 && logger != nil {
				logger.Info("Operation succeeded after retries", zap.Int("retries", i))
			}
			return nil
		}
		
		// 如果是最后一次重试，直接返回错误
		if i == config.MaxRetries {
			break
		}
		
		// 计算下一次重试的间隔
		interval := calculateBackoff(config, i)
		
		// 记录重试日志
		if logger != nil {
			logger.Warn("Operation failed, will retry",
				zap.Int("attempt", i+1),
				zap.Int("max_retries", config.MaxRetries),
				zap.Duration("interval", interval),
				zap.Error(lastErr))
		}
		
		// 等待重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			// 继续重试
		}
	}
	
	return lastErr
}

// Retry 简单重试函数
func Retry(ctx context.Context, logger *zap.Logger, fn RetryableFunc) error {
	return RetryWithConfig(ctx, logger, nil, fn)
}

// calculateBackoff 计算退避时间
func calculateBackoff(config *Config, attempt int) time.Duration {
	// 指数退避算法
	interval := time.Duration(float64(config.MinInterval) * math.Pow(2, float64(attempt)))
	
	// 限制最大间隔
	if interval > config.MaxInterval {
		interval = config.MaxInterval
	}
	
	// 添加随机抖动
	if config.MaxJitter > 0 {
		jitter := time.Duration(rand.Int63n(int64(config.MaxJitter)))
		interval += jitter
	}
	
	return interval
}

