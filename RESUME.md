# 简历可用描述（复制粘贴）

## 项目标题（一行）

轻量 IM 即时通讯服务端 | Go / WebSocket / Redis / MySQL | [GitHub](https://github.com/guyanxi11/im-server)

## 项目描述（推荐贴简历）

基于 Go 实现的即时通讯服务端，覆盖注册登录、WebSocket 长连接、单聊/群聊实时转发、离线消息落库补推与历史消息分页。

- 采用 Hub 模式管理连接：用 channel 串行处理注册/注销，避免 map 并发写；每个连接拆分 readPump/writePump，遵守 WebSocket 不可并发写约束
- 单聊在线非阻塞转发并返回 ACK；离线写入 MySQL Pending，用户上线后按时间序自动补推
- 群聊采用写扩散（fan-out on write）：在线成员即时推送，离线成员落库待补推；群历史单独存主记录，避免成员副本导致查询重复
- 使用 Redis Set 维护在线用户；HTTP 接口基于 JWT Bearer 鉴权，支持单聊/群聊历史分页

## 技术关键词

Golang、WebSocket、并发（goroutine/channel）、JWT、MySQL、Redis、GORM、长连接心跳、离线消息、群聊写扩散
