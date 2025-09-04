package system

import (
	"gin-example/internal/pkg/core"
	"gin-example/internal/repository/mysql"
	"gin-example/internal/repository/redis"

	"go.uber.org/zap"
)

// RegisterHealthRoutes 注册健康检查路由
func RegisterHealthRoutes(logger *zap.Logger, db mysql.Repo, redisRepo *redis.Repo, r core.Mux) {
	h := New(logger, db, redisRepo)
	
	// 注册健康检查路由
	r.Group("").GET("/system/health", h.Health())
}