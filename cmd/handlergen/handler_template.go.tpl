package {{.PackageName}}

import (
	"net/http"
	"strconv"

	"gin-example/internal/code"
	"gin-example/internal/pkg/core"
	"gin-example/internal/repository/mysql"
	"gin-example/internal/repository/mysql/dao"
	"gin-example/internal/repository/mysql/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type handler struct {
	logger  *zap.Logger
	writeDB *dao.Query
	readDB  *dao.Query
}

type genResultInfo struct {
	RowsAffected int64 `json:"rows_affected"`
	Error        error `json:"error"`
}

func New(logger *zap.Logger, db mysql.Repo) *handler {
	return &handler{
		logger:  logger,
		writeDB: dao.Use(db.GetDbW()),
		readDB:  dao.Use(db.GetDbR()),
	}
}

// Create 新增数据
// @Summary 新增数据
// @Description 新增数据
// @Tags Table.{{.VariableName}}
// @Accept json
// @Produce json
// @Param RequestBody body model.{{.StructName}} true "请求参数"
// @Success 200 {object} model.{{.StructName}}
// @Failure 400 {object} code.Failure
// @Router /api/{{.VariableName}} [post]
func (h *handler) Create() core.HandlerFunc {
	return func(ctx core.Context) {
		var createData model.{{.StructName}}
		if err := ctx.ShouldBindJSON(&createData); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		if err := h.writeDB.{{.StructName}}.WithContext(ctx.RequestContext()).Create(&createData); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		ctx.Payload(createData)
	}
}

// List 获取列表数据
// @Summary 获取列表数据
// @Description 获取列表数据
// @Tags Table.{{.VariableName}}
// @Accept json
// @Produce json
// @Success 200 {object} []model.{{.StructName}}
// @Failure 400 {object} code.Failure
// @Router /api/{{.VariableName}}s [get]
func (h *handler) List() core.HandlerFunc {
	return func(ctx core.Context) {
		list, err := h.readDB.{{.StructName}}.WithContext(ctx.RequestContext()).Find()
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		ctx.Payload(list)
	}
}

// GetByID 根据 ID 获取数据
// @Summary 根据 ID 获取数据
// @Description 根据 ID 获取数据
// @Tags Table.{{.VariableName}}
// @Accept json
// @Produce json
// @Param id path string true "ID"
// @Success 200 {object} model.{{.StructName}}
// @Failure 400 {object} code.Failure
// @Router /api/{{.VariableName}}/{id} [get]
func (h *handler) GetByID() core.HandlerFunc {
	return func(ctx core.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		info, err := h.readDB.{{.StructName}}.WithContext(ctx.RequestContext()).Where(h.readDB.{{.StructName}}.ID.Eq(int32(id))).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					"record not found"),
				)
			} else {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					err.Error()),
				)
			}
			return
		}

		ctx.Payload(info)
	}
}

// DeleteByID 根据 ID 删除数据
// @Summary 根据 ID 删除数据
// @Description 根据 ID 删除数据
// @Tags Table.{{.VariableName}}
// @Accept json
// @Produce json
// @Param id path string true "ID"
// @Success 200 {object} genResultInfo
// @Failure 400 {object} code.Failure
// @Router /api/{{.VariableName}}/{id} [delete]
func (h *handler) DeleteByID() core.HandlerFunc {
	return func(ctx core.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		info, err := h.readDB.{{.StructName}}.WithContext(ctx.RequestContext()).Where(h.readDB.{{.StructName}}.ID.Eq(int32(id))).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					"record not found"),
				)
			} else {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					err.Error()),
				)
			}
			return
		}

		result, err := h.writeDB.{{.StructName}}.Delete(info)
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
		}

		resultInfo := new(genResultInfo)
		resultInfo.RowsAffected = result.RowsAffected
		resultInfo.Error = result.Error

		ctx.Payload(resultInfo)
	}
}

// UpdateByID 根据 ID 更新数据
// @Summary 根据 ID 更新数据
// @Description 根据 ID 更新数据
// @Tags Table.{{.VariableName}}
// @Accept json
// @Produce json
// @Param id path string true "ID"
// @Param RequestBody body model.{{.StructName}} true "请求参数"
// @Success 200 {object} genResultInfo
// @Failure 400 {object} code.Failure
// @Router /api/{{.VariableName}}/{id} [put]
func (h *handler) UpdateByID() core.HandlerFunc {
	return func(ctx core.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		var updateData map[string]interface{}
		if err := ctx.ShouldBindJSON(&updateData); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		info, err := h.readDB.{{.StructName}}.WithContext(ctx.RequestContext()).Where(h.readDB.{{.StructName}}.ID.Eq(int32(id))).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					"record not found"),
				)
			} else {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					err.Error()),
				)
			}
			return
		}

		result, err := h.writeDB.{{.StructName}}.WithContext(ctx.RequestContext()).Where(h.writeDB.{{.StructName}}.ID.Eq(info.ID)).Updates(updateData)
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		resultInfo := new(genResultInfo)
		resultInfo.RowsAffected = result.RowsAffected
		resultInfo.Error = result.Error

		ctx.Payload(resultInfo)
	}
}
