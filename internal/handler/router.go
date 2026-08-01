// Package handler 负责注册 HTTP/WS 路由并组装 HTTP 服务
// 作者: wym
package handler

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wym/im-server/internal/config"
	"github.com/wym/im-server/internal/ws"
	"github.com/wym/im-server/pkg/resp"
)

// NewRouter 构建路由表，挂载各业务端点
// db/cfg/rdb 会被注入到各 handler 中；hub 是 WebSocket 连接注册中心，
// 调用方需要在别处启动 hub.Run() 的常驻 goroutine
func NewRouter(db *gorm.DB, cfg *config.Config, rdb *redis.Client, hub *ws.Hub) *http.ServeMux {
	mux := http.NewServeMux()

	authHandler := NewAuthHandler(db, cfg)
	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", authHandler.Login)

	// /ws：WebSocket 接入端点，需要 ?token=xxx 携带登录时签发的 JWT
	mux.HandleFunc("/ws", ws.NewWSHandler(hub, cfg.JWT.Secret))

	// /api/online：调试用途，查看当前 Redis 里记录的在线用户 ID 列表
	// 后续可能会加鉴权限制（仅 admin 可查），当前阶段先开放方便联调
	mux.HandleFunc("/api/online", func(w http.ResponseWriter, r *http.Request) {
		ids, err := hub.OnlineUserIDs(r.Context())
		if err != nil {
			resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询在线用户失败")
			return
		}
		resp.OK(w, map[string]interface{}{"online_user_ids": ids})
	})

	return mux
}

// NewHTTPServer 构造一个 *http.Server，统一在这里设置超时等参数
// 后续接入业务时，超时、读超时需要按长连接场景调整（WebSocket 不宜设过短读超时）
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: h,
	}
}
