# im-server

基于 Go 的轻量 IM（即时通讯）服务端练手项目：JWT 鉴权、WebSocket 长连接、单聊实时转发、离线消息落库补推、历史消息分页、Redis 在线状态。

仓库地址：https://github.com/guyanxi11/im-server

## 功能

- 用户注册 / 登录（bcrypt + JWT）
- WebSocket 接入（`/ws?token=...`）
- Hub 连接管理（上线 / 下线 / 顶替旧连接）
- 单聊实时转发 + ACK 回执（在线 `delivered` / 离线 `saved_offline`）
- 协议层 ping/pong + 应用层 ping
- 离线消息 MySQL 落库，上线自动补推
- 单聊历史消息分页查询（Bearer 鉴权）
- Redis Set 维护在线用户

## 技术栈

Go · gorilla/websocket · GORM · MySQL · Redis · JWT · viper

## 架构示意

```
Client A ──WS──┐
               │     ┌─────────┐     ┌───────┐
Client B ──WS──┼────▶│   Hub   │────▶│ Redis │  (online set)
               │     │ Client  │     └───────┘
HTTP API ──────┘     │ read/   │
                     │ write   │     ┌───────┐
                     │ Pump    │────▶│ MySQL │  (users / messages)
                     └─────────┘     └───────┘
```

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
| GET  | `/api/messages?peer_id=&page=&limit=` | 单聊历史（需 `Authorization: Bearer <token>`） |
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

## 设计要点（面试可讲）

1. **Hub + channel**：`clients` map 只由 `Run()` 单 goroutine 写入，避免并发写 map；外部读用 RWMutex。
2. **读写分离**：每个连接 `readPump` / `writePump` 两个 goroutine；写连接只能走 `send` channel（gorilla/websocket 不允许并发写）。
3. **非阻塞投递**：`SendToUser` 用 `select default`，慢客户端不会拖死发送方。
4. **离线补推**：所有消息落库；上线后在 `register` 写入 map 之后异步 `flushOffline`，保证顺序与可达性。
5. **WS 鉴权放 query**：浏览器原生 WebSocket API 不支持自定义 Header。

## 目录结构

```
cmd/server/          # 入口
internal/
  auth/              # 密码哈希、JWT
  config/            # viper 配置
  db/                # MySQL / Redis 初始化
  handler/           # HTTP 路由、鉴权中间件、历史消息
  model/             # GORM 模型
  store/             # 消息数据访问
  ws/                # Hub / Client / 消息协议
pkg/resp/            # 统一 JSON 响应
```

## 作者

wym
