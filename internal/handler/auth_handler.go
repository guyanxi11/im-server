// Package handler（auth_handler.go）实现用户注册与登录接口
// 作者: wym
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/wym/im-server/internal/auth"
	"github.com/wym/im-server/internal/config"
	"github.com/wym/im-server/internal/model"
	"github.com/wym/im-server/pkg/resp"
)

// 业务错误码：0 表示成功，非 0 按模块分段（1xxx 用户相关），方便前端按 code 分支处理
const (
	CodeInvalidParam   = 1001
	CodeUsernameExists = 1002
	CodeUserNotFound   = 1003
	CodeWrongPassword  = 1004
	CodeServerError    = 1005
)

// AuthHandler 持有数据库连接和配置，提供注册/登录两个 HTTP handler
type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAuthHandler 构造 AuthHandler
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// registerRequest 是注册接口的请求体
type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

// Register 处理 POST /api/register
// 流程：校验参数 -> 检查用户名是否已存在 -> bcrypt 加密密码 -> 插入用户表
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "请求体解析失败")
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" || req.Password == "" {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "用户名和密码不能为空")
		return
	}
	if len(req.Password) < 6 {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "密码长度不能少于 6 位")
		return
	}

	// 用户名唯一性检查：先查一次，虽然存在极小的并发竞态窗口，
	// 但表上有 uniqueIndex 兜底，真正并发冲突时 Create 会报错，这里再兜底处理一次
	var count int64
	if err := h.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "服务器内部错误")
		return
	}
	if count > 0 {
		resp.Fail(w, http.StatusConflict, CodeUsernameExists, "用户名已存在")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "密码加密失败")
		return
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username // 未填昵称时默认用用户名
	}

	user := model.User{
		Username:     req.Username,
		PasswordHash: hashed,
		Nickname:     nickname,
	}
	if err := h.db.Create(&user).Error; err != nil {
		resp.Fail(w, http.StatusConflict, CodeUsernameExists, "用户名已存在")
		return
	}

	resp.OK(w, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
	})
}

// loginRequest 是登录接口的请求体
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 处理 POST /api/login
// 流程：查用户 -> bcrypt 比对密码 -> 签发 JWT
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Fail(w, http.StatusBadRequest, CodeInvalidParam, "请求体解析失败")
		return
	}

	var user model.User
	err := h.db.Where("username = ?", req.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		resp.Fail(w, http.StatusUnauthorized, CodeUserNotFound, "用户名或密码错误")
		return
	}
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "服务器内部错误")
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		// 出于安全考虑，用户不存在和密码错误返回相同的提示，避免暴露"哪个用户名存在"
		resp.Fail(w, http.StatusUnauthorized, CodeWrongPassword, "用户名或密码错误")
		return
	}

	token, err := auth.GenerateToken(h.cfg.JWT.Secret, h.cfg.JWT.ExpireHours, user.ID, user.Username)
	if err != nil {
		resp.Fail(w, http.StatusInternalServerError, CodeServerError, "token 签发失败")
		return
	}

	resp.OK(w, map[string]interface{}{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
	})
}
