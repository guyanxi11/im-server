// Package handler（message_handler.go）实现历史消息查询接口
// 作者: wym
package handler

import (
	"net/http"
	"strconv"

	"github.com/guyanxi11/im-server/internal/store"
	"github.com/guyanxi11/im-server/pkg/resp"
)

// MessageHandler 提供消息相关的 HTTP 接口
type MessageHandler struct {
	msgStore *store.MessageStore
}

// NewMessageHandler 构造 MessageHandler
func NewMessageHandler(msgStore *store.MessageStore) *MessageHandler {
	return &MessageHandler{msgStore: msgStore}
}

// ListHistory 处理 GET /api/messages?peer_id=3&page=1&limit=20
// 需要登录（Bearer token）；只能查"自己和对方"之间的会话，不能偷看别人聊天记录
func (h *MessageHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		resp.Fail(w, http.StatusMethodNotAllowed, CodeInvalidParam, "仅支持 GET")
		return
	}

	userID := userIDFromCtx(r.Context())
	peerID, err := strconv.ParseUint(r.URL.Query().Get("peer_id"), 10, 64)
	if err != nil || peerID == 0 {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "peer_id 必填且必须为正整数")
		return
	}
	if uint(peerID) == userID {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "peer_id 不能是自己")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	list, total, err := h.msgStore.ListHistory(userID, uint(peerID), page, limit)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询历史消息失败")
		return
	}

	// 若 page/limit 被 store 校正过，这里用同样规则回传，方便前端展示
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	resp.OK(w, map[string]interface{}{
		"list":  list,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
