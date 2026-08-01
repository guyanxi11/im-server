// Package model（message.go）定义聊天消息表
// 作者: wym
package model

import "time"

// 消息投递状态
const (
	MsgStatusPending   = 0 // 待投递：接收方当时不在线，上线后需补推
	MsgStatusDelivered = 1 // 已投递：已成功推到接收方 WebSocket
)

// Message 对应 messages 表
// GroupID=0 表示单聊；GroupID>0 表示群聊。
// 单聊：一条消息对应一个 ToUserID。
// 群聊历史：存一条 ToUserID=0 的"会话主记录"供分页查询；
// 群聊离线：给每个离线成员再写一条 ToUserID=成员ID、Status=Pending 的副本，复用现有 flushOffline。
type Message struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	FromUserID   uint       `gorm:"index:idx_from_to;not null" json:"from_user_id"`
	FromUsername string     `gorm:"type:varchar(64);not null" json:"from_username"`
	ToUserID     uint       `gorm:"index:idx_from_to;index:idx_to_status;not null" json:"to_user_id"`
	GroupID      uint       `gorm:"index:idx_group;not null;default:0" json:"group_id"`
	Content      string     `gorm:"type:varchar(2048);not null" json:"content"`
	Status       int        `gorm:"index:idx_to_status;not null;default:0" json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
}

// TableName 显式指定表名
func (Message) TableName() string {
	return "messages"
}
