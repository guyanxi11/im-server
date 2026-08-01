# im-server

基于 Go 的轻量 IM（即时通讯）服务端练手项目：JWT 鉴权、WebSocket 长连接、单聊实时转发、离线消息落库补推、Redis 在线状态。

## 功能

- 用户注册 / 登录（bcrypt + JWT）
- WebSocket 接入（`/ws?token=...`）
- Hub 连接管理（上线 / 下线 / 顶替旧连接）
- 单聊实时转发 + ACK 回执
- 协议层 ping/pong + 应用层 ping
- 离线消息 MySQL 落库，上线自动补推
- Redis Set 维护在线用户

## 技术栈

Go · gorilla/websocket · GORM · MySQL · Redis · JWT · viper

## 快速开始

1. 复制配置并按本地环境修改：

```bash
cp config.example.yaml config.yaml
```

2. 创建数据库：

```sql
CREATE DATABASE IF NOT EXISTS im_server DEFAULT CHARACTER SET utf8mb4;
```

3. 启动 Redis（本机需有 `127.0.0.1:6379`），然后：

```bash
go run ./cmd/server
```

服务默认监听 `:8080`。

## 主要接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册 |
| POST | `/api/login` | 登录，返回 JWT |
| GET  | `/api/online` | 当前在线用户 ID（调试） |
| WS   | `/ws?token=<jwt>` | WebSocket 接入 |

### WebSocket 消息示例

```json
// 客户端发送
{"type":"chat","to":3,"content":"在吗"}

// 对方收到
{"type":"chat","from":2,"from_name":"wym","to":3,"content":"在吗","ts":1735689600}

// 发送方回执（在线 / 离线）
{"type":"ack","ok":true,"msg":"delivered"}
{"type":"ack","ok":true,"msg":"saved_offline"}
```

## 目录结构

```
cmd/server/          # 入口
internal/
  auth/              # 密码哈希、JWT
  config/            # viper 配置
  db/                # MySQL / Redis 初始化
  handler/           # HTTP 路由与鉴权接口
  model/             # GORM 模型
  store/             # 消息数据访问
  ws/                # Hub / Client / 消息协议
pkg/resp/            # 统一 JSON 响应
```

## 作者

wym
