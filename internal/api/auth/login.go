package auth

import (
	"net/http"
	"time"

	"gin-example/internal/code"
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/jwtoken"
	"gin-example/internal/proposal"
	"gin-example/configs"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type handler struct {
	logger *zap.Logger
}

// LoginRequest 登录请求参数
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	UserID    int32  `json:"user_id"`
	Username  string `json:"username"`
}

func New(logger *zap.Logger) *handler {
	return &handler{
		logger: logger,
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录接口
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} code.Failure
// @Router /auth/login [post]
func (h *handler) Login() core.HandlerFunc {
	return func(ctx core.Context) {
		var req LoginRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				err.Error()),
			)
			return
		}

		// 验证用户名密码（示例中简化处理）
		if !h.validateUser(req.Username, req.Password) {
			ctx.AbortWithError(core.Error(
				http.StatusUnauthorized,
				code.JWTAuthVerifyError,
				"用户名或密码错误"),
			)
			return
		}

		// 创建JWT Token
		jwtUtil := jwtoken.New(configs.Get().JWT.Secret)
		
		// 构造用户会话信息
		sessionUserInfo := proposal.SessionUserInfo{
			Id:       1, // 示例用户ID
			UserName: req.Username,
			NickName: req.Username,
		}
		
		// 签发Token（24小时过期）
		tokenString, err := jwtUtil.Sign(sessionUserInfo, 24*time.Hour)
		if err != nil {
			ctx.AbortWithError(core.Error(
				http.StatusInternalServerError,
				code.ServerError,
				"Token生成失败"),
			)
			return
		}

		// 构造响应
		resp := &LoginResponse{
			Token:     tokenString,
			ExpiresAt: time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05"),
			UserID:    sessionUserInfo.Id,
			Username:  sessionUserInfo.UserName,
		}

		ctx.Payload(resp)
	}
}

// validateUser 验证用户（示例实现）
func (h *handler) validateUser(username, password string) bool {
	// 示例中简化处理，实际应查询数据库验证
	if username == "admin" {
		// 比较密码哈希
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		return err == nil
	}
	return false
}