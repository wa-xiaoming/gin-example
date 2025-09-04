package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gin-example/configs"
	"gin-example/internal/pkg/cache"
	"gin-example/internal/pkg/env"
	"gin-example/internal/pkg/logger"
	"gin-example/internal/pkg/registry/etcd"
	"gin-example/internal/pkg/shutdown"
	"gin-example/internal/repository/mysql"
	"gin-example/internal/repository/redis"
	"gin-example/internal/router"

	"go.uber.org/zap"
)

// @title           gin-example API
// @version         1.0
// @description     This is a gin example server.

// @contact.name   xiaoming
// @contact.url    https://github.com/wa-xiaoming
// @contact.email  tony_stake@163.com

// @host      localhost:9999
// @BasePath  /api

// getEnvFromOS 从环境变量或命令行参数获取环境设置
func getEnvFromOS() string {
	// 首先检查环境变量
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}

	// 然后检查命令行参数
	for i, arg := range os.Args {
		if arg == "-env" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "-env=") {
			return strings.TrimPrefix(arg, "-env=")
		}
	}

	return "fat" // 默认环境
}

func main() {
	// 从环境变量或命令行参数获取环境设置
	envName := getEnvFromOS()

	// 设置环境
	env.SetEnv(envName)

	// 初始化日志
	accessLogger, err := logger.NewJSONLogger(
		logger.WithOutputInConsole(),
		logger.WithField("app", configs.ProjectName),
		logger.WithField("environment", envName),
		logger.WithTimeLayout("2006-01-02 15:04:05"),
	)
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = accessLogger.Sync()
	}()

	accessLogger.Info("Starting application", zap.String("environment", envName))

	// 初始化数据库
	accessLogger.Info("Connecting to MySQL...")
	dbRepo, err := mysql.New()
	if err != nil {
		accessLogger.Fatal("Failed to connect to MySQL", zap.Error(err))
		os.Exit(1)
	}

	// 确保数据库连接在程序退出时正确关闭
	shutdown.RegisterHook(func() {
		if err := dbRepo.DbRClose(); err != nil {
			accessLogger.Error("Failed to close read database connection", zap.Error(err))
		}
		if err := dbRepo.DbWClose(); err != nil {
			accessLogger.Error("Failed to close write database connection", zap.Error(err))
		}
		accessLogger.Info("Database connections closed")
	})

	accessLogger.Info("MySQL connected successfully")

	// 初始化Redis
	accessLogger.Info("Connecting to Redis...")
	redisRepo := redis.New()

	// 确保Redis连接在程序退出时正确关闭
	shutdown.RegisterHook(func() {
		if err := redisRepo.Close(); err != nil {
			accessLogger.Error("Failed to close redis connection", zap.Error(err))
		}
		accessLogger.Info("Redis connection closed")
	})

	// 检查Redis连接是否成功
	redisClient := redisRepo.GetClient()
	if redisClient == nil {
		accessLogger.Warn("Failed to get Redis client, using local cache only")
	} else {
		// 尝试ping Redis服务器以确认连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			accessLogger.Warn("Failed to ping Redis server", zap.Error(err))
			redisClient = nil // 置空客户端，后续只使用本地缓存
		} else {
			accessLogger.Info("Redis connected successfully")
		}
	}

	// 创建缓存实例
	var appCache cache.Cache
	if redisClient != nil {
		// 使用多级缓存（本地+Redis）
		appCache = cache.NewMultiLevelCache(redisClient)
		accessLogger.Info("Using multi-level cache (local + Redis)")
	} else {
		// 仅使用本地缓存
		appCache = cache.NewLocalCache(1000, 5*time.Minute)
		accessLogger.Warn("Using local cache only")
	}

	// 初始化服务注册
	accessLogger.Info("Initializing service registry...")
	var serviceRegistry *etcd.Registry
	serviceRegistry, err = etcd.NewRegistry(accessLogger)
	if err != nil {
		accessLogger.Error("Failed to create service registry", zap.Error(err))
	} else {
		// 注册服务
		accessLogger.Info("Registering service...")
		if err := serviceRegistry.Register(); err != nil {
			accessLogger.Error("Failed to register service", zap.Error(err))
		} else {
			accessLogger.Info("Service registered successfully")
		}

		// 优雅关闭时注销服务
		shutdown.RegisterHook(func() {
			accessLogger.Info("Deregistering service...")
			if serviceRegistry != nil {
				if err := serviceRegistry.Deregister(); err != nil {
					accessLogger.Error("Failed to deregister service", zap.Error(err))
				} else {
					accessLogger.Info("Service deregistered successfully")
				}
			}
		})
	}

	// 初始化 HTTP 服务
	accessLogger.Info("Initializing HTTP mux...")
	httpMux, err := router.NewHTTPMux(accessLogger, dbRepo, appCache, &redisRepo)
	if err != nil {
		accessLogger.Fatal("Failed to create HTTP mux", zap.Error(err))
		os.Exit(1)
	}
	accessLogger.Info("HTTP mux initialized successfully")

	server := &http.Server{
		Addr:    configs.ProjectPort,
		Handler: httpMux,
	}

	// 启动HTTP服务
	accessLogger.Info("Starting HTTP server", zap.String("port", configs.ProjectPort))
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			accessLogger.Fatal("HTTP server startup error", zap.Error(err))
			os.Exit(1)
		}
	}()
	accessLogger.Info("HTTP server started successfully")

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	accessLogger.Info("Shutting down server...")

	// 设置超时上下文用于关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		accessLogger.Error("Server shutdown error", zap.Error(err))
	}

	// 执行停机钩子
	shutdown.Close(func() {
		accessLogger.Info("Server shutdown completed")
	})
}
