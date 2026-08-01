// Package handler（middleware.go）提供 HTTP 鉴权中间件
// 作者: wym
package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/guyanxi11/im-server/internal/auth"
	"github.com/guyanxi11/im-server/pkg/resp"
)

// contextKey 是自定义的 context key 类型，避免和第三方包的 string key 冲突
type contextKey string

const (
	// ctxUserIDKey 把 JWT 解析出的 userID 挂到 request context，供下游 handler 读取
	ctxUserIDKey   contextKey = "user_id"
	ctxUsernameKey contextKey = "username"

	CodeUnauthorized = 1006
)

// withJWTAuth 校验 Authorization: Bearer <token>，通过后把 userID/username 写入 context
// 注意：WebSocket 鉴权仍走 query token（浏览器 WS API 不支持自定义 Header），HTTP 业务接口走本中间件
func withJWTAuth(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			resp.Fail(w, http.StatusUnauthorized, CodeUnauthorized, "缺少 Authorization Bearer token")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			resp.Fail(w, http.StatusUnauthorized, CodeUnauthorized, "token 无效或已过期")
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, ctxUsernameKey, claims.Username)
		next(w, r.WithContext(ctx))
	}
}

// userIDFromCtx 从 context 取出当前登录用户 ID；中间件保证调用时一定存在
func userIDFromCtx(ctx context.Context) uint {
	v, _ := ctx.Value(ctxUserIDKey).(uint)
	return v
}
