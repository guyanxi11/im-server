// Package ws（message.go）定�?WebSocket 消息协议
// 作�? wym
//
// 客户�?-> 服务端（上行）：
//   {"type":"chat","to":3,"content":"在吗"}
//   {"type":"ping"}                         // 应用层心跳，可�?//
// 服务�?-> 客户端（下行）：
//   {"type":"chat","from":2,"from_name":"wym","to":3,"content":"在吗","ts":1735689600}
//   {"type":"ack","ok":true,"msg":"delivered"}   // 投递回�?//   {"type":"error","msg":"目标用户不在�?}
//   {"type":"pong"}
package ws

import "encoding/json"

// 消息类型常量
const (
	MsgTypeChat  = "chat"  // 单聊文本消息
	MsgTypePing  = "ping"  // 应用层心跳请�?	MsgTypePong  = "pong"  // 应用层心跳响�?	MsgTypeAck   = "ack"   // 投递回�?	MsgTypeError = "error" // 错误通知
)

// InboundMessage 是客户端发给服务端的上行消息结构
type InboundMessage struct {
	Type    string `json:"type"`
	To      uint   `json:"to"`      // 单聊目标用户 ID
	Content string `json:"content"` // 消息正文
}

// OutboundMessage 是服务端推给客户端的下行消息结构
type OutboundMessage struct {
	Type     string `json:"type"`
	From     uint   `json:"from,omitempty"`
	FromName string `json:"from_name,omitempty"`
	To       uint   `json:"to,omitempty"`
	Content  string `json:"content,omitempty"`
	OK       bool   `json:"ok,omitempty"`
	Msg      string `json:"msg,omitempty"`
	TS       int64  `json:"ts,omitempty"` // Unix 秒级时间�?}

// encodeOutbound 把下行消息序列化�?JSON 字节，供塞入 Client.send
func encodeOutbound(msg OutboundMessage) ([]byte, error) {
	return json.Marshal(msg)
}
