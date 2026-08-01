// Package model（message.go）定义聊天消息表
// 作�? wym
package model

import "time"

// 消息投递状�?const (
	MsgStatusPending   = 0 // 待投递：接收方当时不在线，上线后需补推
	MsgStatusDelivered = 1 // 已投递：已成功推到接收方 WebSocket
)

// Message 对应 messages �?// 设计取舍：所有单聊消息都落库（不只是离线），这样后续历史消息分页可以直接查本表，
// 不必再维护两套存储；FromUsername 做冗余字段，补推离线消息时无需�?JOIN users
type Message struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	FromUserID   uint       `gorm:"index:idx_from;not null" json:"from_user_id"`
	FromUsername string     `gorm:"type:varchar(64);not null" json:"from_username"`
	ToUserID     uint       `gorm:"index:idx_to_status;not null" json:"to_user_id"`
	Content      string     `gorm:"type:varchar(2048);not null" json:"content"`
	Status       int        `gorm:"index:idx_to_status;not null;default:0" json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
}

// TableName 显式指定表名
func (Message) TableName() string {
	return "messages"
}
