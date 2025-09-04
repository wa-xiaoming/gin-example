package auth

import (
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/jwtoken"

	"go.uber.org/zap"
)

// RegisterAuthRoutes 注册认证路由
func RegisterAuthRoutes(logger *zap.Logger, r core.Mux) {
	h := New(logger)
	
	// 创建JWT中间件
	jwtMiddleware := jwtoken.NewJWTAuthMiddleware()
	
	// 注册登录路由（无需认证）
	r.Group("").POST("/auth/login", h.Login())
	
	// 注册需要认证的路由示例
	authGroup := r.Group("/auth")
	authGroup.GET("/profile", jwtMiddleware.Middleware(), func(ctx core.Context) {
		ctx.Payload(map[string]interface{}{
			"message": "认证成功",
		})
	})
}