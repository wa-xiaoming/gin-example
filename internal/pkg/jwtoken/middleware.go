package jwtoken

import (
	"net/http"
	"strings"

	"gin-example/internal/code"
	"gin-example/internal/pkg/core"
	"gin-example/internal/proposal"
	"gin-example/configs"

	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware JWT认证中间件
type JWTAuthMiddleware struct {
	jwtSecret string
}

// NewJWTAuthMiddleware 创建JWT认证中间件
func NewJWTAuthMiddleware() *JWTAuthMiddleware {
	secret := configs.Get().JWT.Secret
	if secret == "" {
		// 如果配置中没有密钥，使用默认值
		secret = "default_secret_key"
	}
	
	return &JWTAuthMiddleware{
		jwtSecret: secret,
	}
}

// Middleware JWT认证中间件函数
func (m *JWTAuthMiddleware) Middleware() core.HandlerFunc {
	return func(ctx core.Context) {
		// 从Header中获取Authorization
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"缺少Authorization头信息",
			))
			return
		}

		// 解析JWT Token
		var tokenString string
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			tokenString = authHeader
		}

		// 验证Token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"Token解析失败: "+err.Error(),
			))
			return
		}

		if !token.Valid {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"无效的Token",
			))
			return
		}

		// 获取Claims
		claims, ok := token.Claims.(*Claims)
		if !ok {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"Token声明解析失败",
			))
			return
		}

		// 将用户信息存储到上下文中
		ctx.Set(_SessionUserInfo, claims.SessionUserInfo)
		ctx.Next()
	}
}

// RBACMiddleware RBAC权限控制中间件
type RBACMiddleware struct {
	rbac RBACInterface
}

// RBACInterface RBAC接口
type RBACInterface interface {
	CheckPermission(ctx core.StdContext, userID int32, resource, action string) (bool, error)
}

// NewRBACMiddleware 创建RBAC权限控制中间件
func NewRBACMiddleware(rbac RBACInterface) *RBACMiddleware {
	return &RBACMiddleware{
		rbac: rbac,
	}
}

// Middleware RBAC权限控制中间件函数
func (m *RBACMiddleware) Middleware(resource, action string) core.HandlerFunc {
	return func(ctx core.Context) {
		// 从上下文中获取用户信息
		userInfoInterface, exists := ctx.Get(_SessionUserInfo)
		if !exists {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"用户未认证",
			))
			return
		}

		userInfo, ok := userInfoInterface.(proposal.SessionUserInfo)
		if !ok {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"用户信息解析失败",
			))
			return
		}

		// 检查权限
		hasPermission, err := m.rbac.CheckPermission(ctx.RequestContext(), userInfo.Id, resource, action)
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusInternalServerError,
				code.ServerError,
				"权限检查失败: "+err.Error(),
			))
			return
		}

		if !hasPermission {
			ctx.AbortWithError(core.Error(
				http.StatusForbidden,
				code.ServerError,
				"权限不足",
			))
			return
		}

		ctx.Next()
	}
}

const _SessionUserInfo = "_session_user_info"