# 简历可用描述（复制粘贴）

## 项目标题（一行）

轻量 IM 即时通讯服务端 | Go / WebSocket / Redis / MySQL | [GitHub](https://github.com/guyanxi11/im-server)

## 项目描述（推荐贴简历）

基于 Go 实现的即时通讯服务端，覆盖注册登录、WebSocket 长连接、单聊实时转发、离线消息落库补推与历史消息分页。

- 采用 Hub 模式管理连接：用 channel 串行处理注册/注销，避免 map 并发写；每个连接拆分 readPump/writePump，遵守 WebSocket 不可并发写约束
- 在线消息经 Hub 非阻塞转发并返回 ACK；离线消息写入 MySQL（status=Pending），用户上线后按时间序自动补推并标记已投递
- 使用 Redis Set 维护在线用户；HTTP 历史接口基于 JWT Bearer 鉴权，按会话双方双向查询并分页返回
- 协议层 ping/pong 与应用层心跳结合，超时断连清理，降低假死连接占用

## 技术关键词

Golang、WebSocket、并发（goroutine/channel）、JWT、MySQL、Redis、GORM、长连接心跳、离线消息
