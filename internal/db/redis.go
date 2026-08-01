// Package db（redis.go）负责初始化 Redis 连接
// 作者: wym
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitRedis 建立 Redis 连接并做一次 PING 探活，失败直接返回 error 让上层决定是否终止启动
func InitRedis(addr, password string, dbIndex int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbIndex,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return client, nil
}
