package admin

import (
	"gin-example/internal/pkg/cache"
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/ratelimit"
	"gin-example/internal/repository/mysql"

	"go.uber.org/zap"
)

func RegisterGeneratedAdminRoutes(logger *zap.Logger, db mysql.Repo, r core.RouterGroup, cache cache.Cache) {
	h := New(logger, db, cache)

	// 创建限流中间件
	rateLimitConfig := ratelimit.DefaultRateLimitConfig()
	rateLimitMiddleware := ratelimit.NewRateLimitMiddleware(rateLimitConfig)
	
	// 新增数据
	r.POST("/admin", rateLimitMiddleware.Middleware(), h.Create())

	// 获取列表数据
	r.GET("/admins", rateLimitMiddleware.Middleware(), h.List())

	// 根据 ID 获取数据
	r.GET("/admin/:id", rateLimitMiddleware.Middleware(), h.GetByID())

	// 根据 ID 更新数据
	r.PUT("/admin/:id", rateLimitMiddleware.Middleware(), h.UpdateByID())

	// 根据 ID 删除数据
	r.DELETE("/admin/:id", rateLimitMiddleware.Middleware(), h.DeleteByID())
}