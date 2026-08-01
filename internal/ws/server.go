// Package ws（server.go）负�?WebSocket 握手升级与鉴�?// 作�? wym
//
// 鉴权方式：token 放在 URL query 参数里，例如 ws://host/ws?token=xxx
// 之所以不�?Authorization 请求头，是因为浏览器原生 WebSocket API（new WebSocket(url)�?// 不支持自定义请求头，这是前端 WS 鉴权的通行做法（业界如此，不是权宜之计�?package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/guyanxi11/im-server/internal/auth"
)

// upgrader 负责�?HTTP 连接升级�?WebSocket 连接
// CheckOrigin 这里放开，方便本地用浏览�?工具测试；生产环境应校验来源域名
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewWSHandler 构�?/ws 端点的处理函�?// hub：连接注册中心；jwtSecret：用于校�?URL 里携带的 token
func NewWSHandler(hub *Hub, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// 鉴权通过后才升级连接：未鉴权的请求不应该占用一�?WebSocket 连接资源
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] upgrade failed: %v", err)
			return
		}

		client := &Client{
			hub:      hub,
			conn:     conn,
			UserID:   claims.UserID,
			Username: claims.Username,
			send:     make(chan []byte, 256),
		}

		hub.register <- client

		// writePump 独立 goroutine 运行；readPump 阻塞在当�?goroutine�?		// 直到连接断开（http handler �?goroutine 生命周期正好和这条连接绑定）
		go client.writePump()
		client.readPump()
	}
}
