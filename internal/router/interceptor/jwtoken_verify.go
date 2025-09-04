package interceptor

import (
	"fmt"
	"net/http"

	"gin-example/configs"
	"gin-example/internal/code"
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/jwtoken"
	"gin-example/internal/proposal"
)

func (i *Interceptor) JWTokenAuthVerify(ctx core.Context) (sessionUserInfo proposal.SessionUserInfo, err core.BusinessError) {
	// 具体 Header 参数，可根据实际情况调整
	headerAuthorizationString := ctx.GetHeader("Authorization")
	if headerAuthorizationString == "" {
		err = core.Error(
			http.StatusUnauthorized,
			code.JWTAuthVerifyError,
			"Header 中缺少 Authorization 参数")

		return
	}

	// 验证 JWT 是否合法
	jwtClaims, jwtErr := jwtoken.New(configs.Get().JWT.Secret).Parse(headerAuthorizationString)
	if jwtErr != nil {
		err = core.Error(
			http.StatusUnauthorized,
			code.JWTAuthVerifyError,
			fmt.Sprintf("jwt token 验证失败： %s", jwtErr.Error()))

		return
	}

	sessionUserInfo = jwtClaims.SessionUserInfo

	return
}
