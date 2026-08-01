// Package auth（jwt.go）负�?JWT 的签发与解析
// 作�? wym
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是自定义�?JWT 载荷，除标准字段外附�?UserID/Username
// 后续 WebSocket 鉴权、单�?群聊消息路由都靠 UserID 识别身份
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// ErrInvalidToken 表示 token 缺失/格式错误/签名不匹�?已过期等所有解析失败情况的统一错误
var ErrInvalidToken = errors.New("invalid or expired token")

// GenerateToken 签发一�?JWT，有效期 expireHours 小时
func GenerateToken(secret string, expireHours int, userID uint, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 校验并解�?token，成功返�?Claims（包�?UserID/Username�?func ParseToken(secret string, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// 显式校验签名算法，防�?algorithm none"之类的伪�?token 攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
