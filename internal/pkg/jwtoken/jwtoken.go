package jwtoken

import (
	"time"

	"gin-example/internal/proposal"

	"github.com/golang-jwt/jwt/v5"
)

var _ Token = (*token)(nil)

type Token interface {
	i()
	Sign(jwtInfo proposal.SessionUserInfo, expireDuration time.Duration) (tokenString string, err error)
	Parse(tokenString string) (*Claims, error)
}

type token struct {
	secret string
}

// Claims JWT声明结构
type Claims struct {
	proposal.SessionUserInfo
	jwt.RegisteredClaims
}

func New(secret string) Token {
	return &token{
		secret: secret,
	}
}

func (t *token) i() {}

func (t *token) Sign(sessionUserInfo proposal.SessionUserInfo, expireDuration time.Duration) (tokenString string, err error) {
	// 检查密钥是否为空
	if t.secret == "" {
		return "", jwt.ErrInvalidKey
	}

	claims := Claims{
		sessionUserInfo,
		jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
		},
	}

	tokenString, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(t.secret))
	return
}

func (t *token) Parse(tokenString string) (*Claims, error) {
	// 检查密钥是否为空
	if t.secret == "" {
		return nil, jwt.ErrInvalidKey
	}

	// 检查token字符串是否为空
	if tokenString == "" {
		return nil, jwt.ErrTokenMalformed
	}

	tokenClaims, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(t.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, jwt.ErrTokenInvalidClaims
}