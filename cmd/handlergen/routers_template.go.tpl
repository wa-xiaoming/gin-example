package {{.PackageName}}

import (
	"gin-example/internal/pkg/core"
	"gin-example/internal/repository/mysql"

	"go.uber.org/zap"
)

func RegisterGenerated{{.StructName}}Routes(logger *zap.Logger, db mysql.Repo, r core.RouterGroup) {
	h := New(logger, db)

	// 新增数据
	r.POST("/{{.PackageName}}", h.Create())

	// 获取列表数据
	r.GET("/{{.PackageName}}s", h.List())

	// 根据 ID 获取数据
	r.GET("/{{.PackageName}}/:id", h.GetByID())

	// 根据 ID 更新数据
	r.PUT("/{{.PackageName}}/:id", h.UpdateByID())

	// 根据 ID 删除数据
	r.DELETE("/{{.PackageName}}/:id", h.DeleteByID())
}
