// Package db 负责初始化数据库连接
// 作者: wym
package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wym/im-server/internal/model"
)

// InitMySQL 建立 MySQL 连接，并对已注册的模型执行自动迁移（建表/加字段）
// 注意：AutoMigrate 只适合开发阶段快速迭代，生产环境应改用显式 SQL 迁移脚本
// （这一点和 DMIPSS 项目"新表只追加到 sql.sql"的规范思路一致，只是当前是练手项目先用 AutoMigrate 简化流程）
func InitMySQL(dsn string) (*gorm.DB, error) {
	dbConn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // 只打印慢查询/警告/错误，避免每条 SQL 都刷屏
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	// 自动迁移：根据 model 定义创建/更新表结构
	if err := dbConn.AutoMigrate(
		&model.User{},
		&model.Message{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移表结构失败: %w", err)
	}

	return dbConn, nil
}
