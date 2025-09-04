package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gin-example/internal/code"
	"gin-example/internal/pkg/cache"
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
	cache   cache.Cache
}

type genResultInfo struct {
	RowsAffected int64 `json:"rows_affected"`
	Error        error `json:"error"`
}

func New(logger *zap.Logger, db mysql.Repo, cache cache.Cache) *handler {
	return &handler{
		logger:  logger,
		writeDB: dao.Use(db.GetDbW()),
		readDB:  dao.Use(db.GetDbR()),
		cache:   cache,
	}
}

// Create 新增数据
// @Summary 新增数据
// @Description 新增数据
// @Tags Table.admin
// @Accept json
// @Produce json
// @Param RequestBody body model.Admin true "请求参数"
// @Success 200 {object} model.Admin
// @Failure 400 {object} code.Failure
// @Router /api/admin [post]
func (h *handler) Create() core.HandlerFunc {
	return func(ctx core.Context) {
		var createData model.Admin
		if err := ctx.ShouldBindJSON(&createData); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		if err := h.writeDB.Admin.WithContext(ctx.RequestContext()).Create(&createData); err != nil {
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
// @Tags Table.admin
// @Accept json
// @Produce json
// @Success 200 {object} []model.Admin
// @Failure 400 {object} code.Failure
// @Router /api/admins [get]
func (h *handler) List() core.HandlerFunc {
	return func(ctx core.Context) {
		// 尝试从缓存获取
		var list []*model.Admin
		cacheKey := "admin:list"
		
		if exists, _ := h.cache.Exists(cacheKey); exists {
			if err := h.cache.Get(cacheKey, &list); err == nil {
				ctx.Payload(list)
				return
			}
		}

		// 缓存未命中，从数据库获取
		var err error
		list, err = h.readDB.Admin.WithContext(ctx.RequestContext()).Find()
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		// 存储到缓存，过期时间5分钟
		_ = h.cache.Set(cacheKey, list, 5*time.Minute)

		ctx.Payload(list)
	}
}

// GetByID 根据 ID 获取数据
// @Summary 根据 ID 获取数据
// @Description 根据 ID 获取数据
// @Tags Table.admin
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} model.Admin
// @Failure 400 {object} code.Failure
// @Router /api/admin/{id} [get]
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

		// 尝试从缓存获取
		var data *model.Admin
		cacheKey := fmt.Sprintf("admin:%d", id)
		
		if exists, _ := h.cache.Exists(cacheKey); exists {
			if err := h.cache.Get(cacheKey, &data); err == nil {
				ctx.Payload(data)
				return
			}
		}

		// 缓存未命中，从数据库获取
		data, err = h.readDB.Admin.WithContext(ctx.RequestContext()).Where(h.readDB.Admin.ID.Eq(int32(id))).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.ServerError,
					"data not found"),
				)
				return
			}

			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				err.Error()),
			)
			return
		}

		// 存储到缓存，过期时间10分钟
		_ = h.cache.Set(cacheKey, data, 10*time.Minute)

		ctx.Payload(data)
	}
}

// UpdateByID 根据 ID 更新数据
// @Summary 根据 ID 更新数据
// @Description 根据 ID 更新数据
// @Tags Table.admin
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param RequestBody body model.Admin true "请求参数"
// @Success 200 {object} genResultInfo
// @Failure 400 {object} code.Failure
// @Router /api/admin/{id} [put]
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

		var updateData model.Admin
		if err := ctx.ShouldBindJSON(&updateData); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		resultInfo := new(genResultInfo)
		result, err := h.writeDB.Admin.WithContext(ctx.RequestContext()).Where(h.writeDB.Admin.ID.Eq(int32(id))).Updates(updateData)
		if err != nil {
			resultInfo.Error = err
		} else {
			resultInfo.RowsAffected = result.RowsAffected
		}
		
		if resultInfo.Error != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				resultInfo.Error.Error()),
			)
			return
		}

		// 删除相关缓存
		cacheKey := fmt.Sprintf("admin:%d", id)
		_ = h.cache.Delete(cacheKey)
		_ = h.cache.Delete("admin:list")

		ctx.Payload(resultInfo)
	}
}

// DeleteByID 根据 ID 删除数据
// @Summary 根据 ID 删除数据
// @Description 根据 ID 删除数据
// @Tags Table.admin
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} genResultInfo
// @Failure 400 {object} code.Failure
// @Router /api/admin/{id} [delete]
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

		resultInfo := new(genResultInfo)
		result, err := h.writeDB.Admin.WithContext(ctx.RequestContext()).Where(h.writeDB.Admin.ID.Eq(int32(id))).Delete()
		if err != nil {
			resultInfo.Error = err
		} else {
			resultInfo.RowsAffected = result.RowsAffected
		}
		
		if resultInfo.Error != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ServerError,
				resultInfo.Error.Error()),
			)
			return
		}

		// 删除相关缓存
		cacheKey := fmt.Sprintf("admin:%d", id)
		_ = h.cache.Delete(cacheKey)
		_ = h.cache.Delete("admin:list")

		ctx.Payload(resultInfo)
	}
}