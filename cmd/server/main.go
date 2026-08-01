// Package main 是 im-server 的程序入口
// 作者: wym
// 启动流程：加载配置 -> 连接 MySQL（自动迁移建表） -> 连接 Redis -> 启动 Hub -> 注册路由 -> 启动 HTTP 服务
package main

import (
	"log"

	"github.com/guyanxi11/im-server/internal/config"
	"github.com/guyanxi11/im-server/internal/db"
	"github.com/guyanxi11/im-server/internal/handler"
	"github.com/guyanxi11/im-server/internal/store"
	"github.com/guyanxi11/im-server/internal/ws"
)

func main() {
	cfg := config.Load()

	dbConn, err := db.InitMySQL(cfg.MySQL.DSN)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	log.Println("MySQL 连接成功，表结构已自动迁移")

	rdb, err := db.InitRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}
	log.Println("Redis 连接成功")

	msgStore := store.NewMessageStore(dbConn)

	// Hub 是全局唯一的连接注册中心，Run() 必须常驻运行，不能跟随某个请求的生命周期
	hub := ws.NewHub(rdb, msgStore)
	go hub.Run()

	mux := handler.NewRouter(dbConn, cfg, rdb, hub)
	srv := handler.NewHTTPServer(cfg.Server.Addr, mux)

	log.Printf("im-server listening on %s", cfg.Server.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server exit: %v", err)
	}
}
