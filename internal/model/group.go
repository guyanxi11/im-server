// Package model（group.go）定义群与群成员表
// 作者: wym
package model

import "time"

// Group 对应 groups 表
type Group struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(64);not null" json:"name"`
	OwnerID   uint      `gorm:"index;not null" json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 显式指定表名
func (Group) TableName() string {
	return "groups"
}

// GroupMember 对应 group_members 表
// 联合唯一索引保证同一用户不会重复加入同一群
type GroupMember struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	GroupID  uint      `gorm:"uniqueIndex:uk_group_user;not null" json:"group_id"`
	UserID   uint      `gorm:"uniqueIndex:uk_group_user;index;not null" json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

// TableName 显式指定表名
func (GroupMember) TableName() string {
	return "group_members"
}
