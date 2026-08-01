// Package resp 提供统一的 HTTP JSON 响应格式
// 作者: wym
// 约定：{"code": 0, "message": "ok", "data": ...}；code 非 0 表示业务错误
package resp

import (
	"encoding/json"
	"net/http"
)

// Body 是统一响应体结构
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 返回成功响应，HTTP 状态码固定 200，业务 code 固定 0
func OK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// Fail 返回业务失败响应；httpStatus 控制 HTTP 状态码，code/message 描述具体业务错误
// 例如：参数错误用 400 + code=1001，未授权用 401 + code=1002
func Fail(w http.ResponseWriter, httpStatus int, code int, message string) {
	writeJSON(w, httpStatus, Body{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, httpStatus int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(body)
}
