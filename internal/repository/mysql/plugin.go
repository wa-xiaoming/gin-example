package mysql

import (
	"database/sql"
	"strconv"
	"time"

	metrics "gin-example/internal/metrics"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	callBackBeforeName = "core:before"
	callBackAfterName  = "core:after"
	startTime          = "_start_time"
)

// TracePlugin 数据库追踪插件
type TracePlugin struct {
	logger *zap.Logger
}

func (op *TracePlugin) Name() string {
	return "TracePlugin"
}

func (op *TracePlugin) Initialize(db *gorm.DB) error {
	// 注册回调函数
	_ = db.Callback().Create().After("gorm:create").Register("trace:create", op.traceCreate)
	_ = db.Callback().Query().After("gorm:query").Register("trace:query", op.traceQuery)
	_ = db.Callback().Update().After("gorm:update").Register("trace:update", op.traceUpdate)
	_ = db.Callback().Delete().After("gorm:delete").Register("trace:delete", op.traceDelete)
	
	// 启动连接池监控
	go op.monitorConnectionPool(db)
	
	return nil
}

var _ gorm.Plugin = &TracePlugin{}

// traceCreate 创建操作追踪
func (op *TracePlugin) traceCreate(db *gorm.DB) {
	if db.Error != nil {
		metrics.RecordError("db:create", db.Error.Error())
	}
	
	metrics.RecordDBQuery("write", "create", strconv.FormatBool(db.Error == nil))
	
	// 记录执行时间
	// 这里可以从上下文中获取开始时间并计算耗时
}

// traceQuery 查询操作追踪
func (op *TracePlugin) traceQuery(db *gorm.DB) {
	if db.Error != nil {
		metrics.RecordError("db:query", db.Error.Error())
	}
	
	readWrite := "read"
	if db.Dialector.Name() == "mysql" {
		// 根据SQL语句判断是读还是写操作
		// 这里简化处理，默认认为查询是读操作
	}
	
	metrics.RecordDBQuery(readWrite, "query", strconv.FormatBool(db.Error == nil))
}

// traceUpdate 更新操作追踪
func (op *TracePlugin) traceUpdate(db *gorm.DB) {
	if db.Error != nil {
		metrics.RecordError("db:update", db.Error.Error())
	}
	
	metrics.RecordDBQuery("write", "update", strconv.FormatBool(db.Error == nil))
}

// traceDelete 删除操作追踪
func (op *TracePlugin) traceDelete(db *gorm.DB) {
	if db.Error != nil {
		metrics.RecordError("db:delete", db.Error.Error())
	}
	
	metrics.RecordDBQuery("write", "delete", strconv.FormatBool(db.Error == nil))
}

// monitorConnectionPool 监控连接池状态
func (op *TracePlugin) monitorConnectionPool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats := sqlDB.Stats()
			
			// 记录连接池指标
			metrics.SetDBConnections("total", "open", float64(stats.OpenConnections))
			metrics.SetDBConnections("total", "in_use", float64(stats.InUse))
			metrics.SetDBConnections("total", "idle", float64(stats.Idle))
			
			// 记录等待次数
			metrics.RecordDBQuery("connection", "wait_count", strconv.FormatBool(stats.WaitCount == 0))
			
		}
	}
}

// ConfigureConnectionPool 配置数据库连接池
func ConfigureConnectionPool(db *sql.DB, config *ConnectionPoolConfig) {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}
	
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

// DefaultConnectionPoolConfig 默认连接池配置
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}
}