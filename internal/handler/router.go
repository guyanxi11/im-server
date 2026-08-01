// Package handler 负责注册 HTTP/WS 路由并组装 HTTP 服务
// 作者: wym
package handler

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/guyanxi11/im-server/internal/config"
	"github.com/guyanxi11/im-server/internal/store"
	"github.com/guyanxi11/im-server/internal/ws"
	"github.com/guyanxi11/im-server/pkg/resp"
)

// NewRouter 构建路由表，挂载各业务端点
func NewRouter(
	db *gorm.DB,
	cfg *config.Config,
	rdb *redis.Client,
	hub *ws.Hub,
	msgStore *store.MessageStore,
	groupStore *store.GroupStore,
) *http.ServeMux {
	mux := http.NewServeMux()

	authHandler := NewAuthHandler(db, cfg)
	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", authHandler.Login)

	mux.HandleFunc("/ws", ws.NewWSHandler(hub, cfg.JWT.Secret))

	mux.HandleFunc("/api/online", func(w http.ResponseWriter, r *http.Request) {
		ids, err := hub.OnlineUserIDs(r.Context())
		if err != nil {
			resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询在线用户失败")
			return
		}
		resp.OK(w, map[string]interface{}{"online_user_ids": ids})
	})

	msgHandler := NewMessageHandler(msgStore)
	mux.HandleFunc("/api/messages", withJWTAuth(cfg.JWT.Secret, msgHandler.ListHistory))

	// 群相关：用不同路径区分方法语义，统一要求登录
	groupHandler := NewGroupHandler(groupStore, msgStore)
	mux.HandleFunc("/api/groups", withJWTAuth(cfg.JWT.Secret, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			groupHandler.Create(w, r)
		case http.MethodGet:
			groupHandler.ListMine(w, r)
		default:
			resp.Fail(w, http.StatusMethodNotAllowed, CodeInvalidParam, "仅支持 GET/POST")
		}
	}))
	mux.HandleFunc("/api/groups/members", withJWTAuth(cfg.JWT.Secret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			resp.Fail(w, http.StatusMethodNotAllowed, CodeInvalidParam, "仅支持 POST")
			return
		}
		groupHandler.AddMembers(w, r)
	}))
	mux.HandleFunc("/api/groups/messages", withJWTAuth(cfg.JWT.Secret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			resp.Fail(w, http.StatusMethodNotAllowed, CodeInvalidParam, "仅支持 GET")
			return
		}
		groupHandler.ListMessages(w, r)
	}))

	return mux
}

// NewHTTPServer 构造一个 *http.Server
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: h,
	}
}
