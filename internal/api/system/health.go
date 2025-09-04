package system

import (
	"context"
	"time"

	"gin-example/internal/pkg/core"
	"gin-example/internal/repository/mysql"
	redisRepo "gin-example/internal/repository/redis"

	"go.uber.org/zap"
)

type handler struct {
	logger *zap.Logger
	db     mysql.Repo
	redis  *redisRepo.Repo
}

func New(logger *zap.Logger, db mysql.Repo, redisRepo *redisRepo.Repo) *handler {
	return &handler{
		logger: logger,
		db:     db,
		redis:  redisRepo,
	}
}

// Health 健康检查
// @Summary 健康检查
// @Description 健康检查
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /system/health [get]
func (h *handler) Health() core.HandlerFunc {
	return func(ctx core.Context) {
		resp := &HealthResponse{
			Status:      "ok",
			Service:     "gin-example",
			Environment: ctx.GetHeader("Environment"),
		}

		// 检查数据库连接
		dbStatus := h.checkDB()
		resp.Components.Database = ComponentStatus{
			Status:  dbStatus.Status,
			Message: dbStatus.Message,
		}

		// 检查Redis连接
		redisStatus := h.checkRedis()
		resp.Components.Redis = ComponentStatus{
			Status:  redisStatus.Status,
			Message: redisStatus.Message,
		}

		// 设置整体状态
		if dbStatus.Status != "up" || redisStatus.Status != "up" {
			resp.Status = "degraded"
		}

		ctx.Payload(resp)
	}
}

// checkDB 检查数据库连接
func (h *handler) checkDB() ComponentStatus {
	status := ComponentStatus{Status: "up"}

	// 测试数据库连接
	dbw := h.db.GetDbW()
	dbr := h.db.GetDbR()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查写连接
	if err := dbw.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		status.Status = "down"
		status.Message = "Database write connection failed: " + err.Error()
		return status
	}

	// 检查读连接
	if err := dbr.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		status.Status = "down"
		status.Message = "Database read connection failed: " + err.Error()
		return status
	}

	status.Message = "Database connections are healthy"
	return status
}

// checkRedis 检查Redis连接
func (h *handler) checkRedis() ComponentStatus {
	status := ComponentStatus{Status: "up"}

	// 获取Redis客户端
	redisClient := (*h.redis).GetClient()
	if redisClient == nil {
		status.Status = "down"
		status.Message = "Redis client is not initialized"
		return status
	}

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		status.Status = "down"
		status.Message = "Redis connection failed: " + err.Error()
		return status
	}

	status.Message = "Redis connection is healthy"
	return status
}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status      string         `json:"status"`
	Service     string         `json:"service"`
	Environment string         `json:"environment"`
	Components  ComponentsInfo `json:"components"`
}

// ComponentsInfo 组件信息
type ComponentsInfo struct {
	Database ComponentStatus `json:"database"`
	Redis    ComponentStatus `json:"redis"`
}

// ComponentStatus 组件状态
type ComponentStatus struct {
	Status  string `json:"status"`  // up, down, degraded
	Message string `json:"message"`
}