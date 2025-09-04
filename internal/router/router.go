package router

import (
	"gin-example/internal/api/admin"
	"gin-example/internal/api/auth"
	"gin-example/internal/api/system"
	"gin-example/internal/pkg/cache"
	"gin-example/internal/pkg/core"
	"gin-example/internal/repository/mysql"
	"gin-example/internal/repository/redis"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func NewHTTPMux(logger *zap.Logger, db mysql.Repo, cache cache.Cache, redisRepo *redis.Repo) (core.Mux, error) {
	if logger == nil {
		return nil, errors.New("logger required")
	}

	if db == nil {
		return nil, errors.New("db required")
	}

	if cache == nil {
		return nil, errors.New("cache required")
	}

	if redisRepo == nil {
		return nil, errors.New("redis required")
	}

	mux, err := core.New(logger,
		core.WithEnableCors(),
		core.WithEnableSwagger(),
		core.WithEnablePProf(),
	)

	if err != nil {
		panic(err)
	}

	// 注册系统路由（包括健康检查）
	system.RegisterHealthRoutes(logger, db, redisRepo, mux)

	// 注册认证路由
	auth.RegisterAuthRoutes(logger, mux)

	// 定义自动生成的路由组前缀为 /api
	generatedRouterGroup := mux.Group("/api")
	// 注意：由于RouterGroup接口没有Use方法，中间件需要在具体的路由中添加

	// 注册路由
	admin.RegisterGeneratedAdminRoutes(logger, db, generatedRouterGroup, cache)

	return mux, nil
}
