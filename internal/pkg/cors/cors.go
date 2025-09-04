package cors

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginCors "github.com/rs/cors/wrapper/gin"
)

func New() gin.HandlerFunc {
	return ginCors.New(ginCors.Options{
		// AllowedOrigins 允许的来源（域名），可以是一个具体的域名或使用通配符 "*" 表示允许所有来源。
		// []string{"http://example.com", "http://another-domain.com"},
		AllowedOrigins: []string{"*"},

		// AllowedMethods 允许的 HTTP 方法，例如 "GET"、"POST"、"PUT" 等。
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},

		// AllowedHeaders 允许的请求标头，例如 "Authorization"、"Content-Type" 等。
		AllowedHeaders: []string{"*"},

		// MaxAge 预检请求的最大缓存时间（秒），用于减少预检请求的频率。
		MaxAge: 86400,

		// AllowCredentials 是否允许携带身份凭证（如 Cookie）。
		AllowCredentials: true,

		// OptionsPassthrough 是否将 OPTIONS 请求传递给下一个处理函数，设置为 true 可以在 Gin 的路由中使用 OPTIONS 请求处理函数。
		OptionsPassthrough: true,
	})
}
