// Package model 定义与数据库表对应的 GORM 模型
// 作者: wym
package model

import "time"

// User 对应 users 表
// 密码字段存 bcrypt 哈希，绝不存明文
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"type:varchar(128);not null" json:"-"` // json:"-" 确保任何情况下都不会被序列化返回给前端
	Nickname     string    `gorm:"type:varchar(64)" json:"nickname"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 显式指定表名，避免 GORM 默认复数化规则产生歧义
func (User) TableName() string {
	return "users"
}
