// Package handler（group_handler.go）实现群相关 HTTP 接口
// 作者: wym
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/guyanxi11/im-server/internal/store"
	"github.com/guyanxi11/im-server/pkg/resp"
)

// GroupHandler 提供建群、拉人、查群、群历史等接口
type GroupHandler struct {
	groupStore *store.GroupStore
	msgStore   *store.MessageStore
}

// NewGroupHandler 构造 GroupHandler
func NewGroupHandler(groupStore *store.GroupStore, msgStore *store.MessageStore) *GroupHandler {
	return &GroupHandler{groupStore: groupStore, msgStore: msgStore}
}

type createGroupRequest struct {
	Name      string `json:"name"`
	MemberIDs []uint `json:"member_ids"`
}

// Create 处理 POST /api/groups
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "请求体解析失败")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "群名称不能为空")
		return
	}

	ownerID := userIDFromCtx(r.Context())
	g, err := h.groupStore.CreateGroup(ownerID, req.Name, req.MemberIDs)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "创建群失败")
		return
	}
	resp.OK(w, g)
}

// ListMine 处理 GET /api/groups —— 列出我加入的群
func (h *GroupHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	list, err := h.groupStore.ListGroupsByUser(userID)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询群列表失败")
		return
	}
	resp.OK(w, map[string]interface{}{"list": list})
}

// AddMembers 处理 POST /api/groups/members —— body: {group_id, member_ids}
// 用 query/body 传 group_id，避免依赖 Go 1.22 的路径变量，保持路由简单
func (h *GroupHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID   uint   `json:"group_id"`
		MemberIDs []uint `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "请求体解析失败")
		return
	}
	if req.GroupID == 0 || len(req.MemberIDs) == 0 {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "group_id 与 member_ids 必填")
		return
	}

	userID := userIDFromCtx(r.Context())
	ok, err := h.groupStore.IsMember(req.GroupID, userID)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "校验成员失败")
		return
	}
	if !ok {
		resp.Fail(w, http.StatusForbidden, CodeUnauthorized, "你不是该群成员，无法拉人")
		return
	}

	if _, err := h.groupStore.GetGroup(req.GroupID); err != nil {
		if errors.Is(err, store.ErrGroupNotFound) {
			resp.Fail(w, http.StatusNotFound, CodeInvalidParam, "群不存在")
			return
		}
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询群失败")
		return
	}

	if err := h.groupStore.AddMembers(req.GroupID, req.MemberIDs); err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "添加成员失败")
		return
	}
	resp.OK(w, map[string]interface{}{"group_id": req.GroupID, "added": req.MemberIDs})
}

// ListMessages 处理 GET /api/groups/messages?group_id=1&page=1&limit=20
func (h *GroupHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	groupID, err := strconv.ParseUint(r.URL.Query().Get("group_id"), 10, 64)
	if err != nil || groupID == 0 {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "group_id 必填")
		return
	}

	ok, err := h.groupStore.IsMember(uint(groupID), userID)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "校验成员失败")
		return
	}
	if !ok {
		resp.Fail(w, http.StatusForbidden, CodeUnauthorized, "你不是该群成员")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, total, err := h.msgStore.ListGroupHistory(uint(groupID), page, limit)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "查询群历史失败")
		return
	}
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
