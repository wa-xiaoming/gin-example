package redis

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"gin-example/configs"
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/timeutil"
	"gin-example/internal/pkg/trace"

	redisV8 "github.com/go-redis/redis/v8"
)

var (
	client *redisV8.Client
	once   sync.Once
)

type Repo interface {
	GetClient() *redisV8.Client
	Close() error
}

type redisRepo struct {
	client *redisV8.Client
}

type loggingHook struct {
	ts time.Time
}

func (h *loggingHook) BeforeProcess(ctx context.Context, cmd redisV8.Cmder) (context.Context, error) {
	h.ts = time.Now()
	return ctx, nil
}

func (h *loggingHook) AfterProcess(ctx context.Context, cmd redisV8.Cmder) error {
	// 先尝试转换为core.StdContext
	monoCtx, ok := ctx.(core.StdContext)
	if !ok {
		// 如果转换失败，记录基本信息到标准日志
		// 这样可以避免在非HTTP请求上下文中使用redis时报错，同时保留有用的调试信息
		fmt.Printf("Redis command executed outside HTTP context: %s\n", cmd.String())
		return nil
	}

	if monoCtx.Trace != nil {
		redisInfo := new(trace.Redis)
		redisInfo.Time = timeutil.CSTLayoutString()
		redisInfo.Stack = fileWithLineNum()
		redisInfo.Cmd = cmd.String()
		redisInfo.CostSeconds = time.Since(h.ts).Seconds()

		monoCtx.Trace.AppendRedis(redisInfo)
	}

	return nil
}

func (h *loggingHook) BeforeProcessPipeline(ctx context.Context, cmds []redisV8.Cmder) (context.Context, error) {
	return ctx, nil
}

func (h *loggingHook) AfterProcessPipeline(ctx context.Context, cmds []redisV8.Cmder) error {
	return nil
}

// New 创建Redis仓库实例
func New() Repo {
	once.Do(func() {
		cfg := configs.Get().Redis
		client = redisV8.NewClient(&redisV8.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Pass,
			DB:           cfg.Db,
			MaxRetries:   3,
			DialTimeout:  time.Second * 5,
			ReadTimeout:  time.Second * 20,
			WriteTimeout: time.Second * 20,
			PoolSize:     50,
			MinIdleConns: 2,
			PoolTimeout:  time.Minute,
		})

		// 增加重试机制确保连接成功
		var err error
		for i := 0; i < 3; i++ {
			err = client.Ping(context.Background()).Err()
			if err == nil {
				break
			}
			time.Sleep(time.Duration(i+1) * time.Second) // 逐步增加等待时间
		}
		
		if err != nil {
			// 记录错误但不panic，允许服务在没有Redis的情况下运行
			fmt.Printf("Warning: Failed to connect to Redis after 3 attempts: %v\n", err)
		}

		client.AddHook(&loggingHook{})
	})

	return &redisRepo{
		client: client,
	}
}

func (r *redisRepo) GetClient() *redisV8.Client {
	return r.client
}

func (r *redisRepo) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func fileWithLineNum() string {
	_, file, line, ok := runtime.Caller(5)
	if ok {
		return file + ":" + strconv.FormatInt(int64(line), 10)
	}

	return ""
}