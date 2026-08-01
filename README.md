# im-server

基于 Go 的轻量 IM（即时通讯）服务端练手项目：JWT 鉴权、WebSocket 长连接、单聊/群聊、离线消息落库补推、历史消息分页、Redis 在线状态。

仓库地址：https://github.com/guyanxi11/im-server

## 功能

- 用户注册 / 登录（bcrypt + JWT）
- WebSocket 接入（`/ws?token=...`）
- Hub 连接管理（上线 / 下线 / 顶替旧连接）
- 单聊实时转发 + ACK 回执（在线 `delivered` / 离线 `saved_offline`）
- 群聊：建群、拉人、写扩散（fan-out）、群历史分页
- 协议层 ping/pong + 应用层 ping
- 离线消息 MySQL 落库，上线自动补推（单聊/群聊统一）
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
                     │ Pump    │────▶│ MySQL │  (users/messages/groups)
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

服务默认监听 `:8080`。浏览器打开 http://localhost:8080 即可使用验收前端。

## 主要接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册 |
| POST | `/api/login` | 登录，返回 JWT |
| GET  | `/api/online` | 当前在线用户 ID（调试） |
| GET  | `/api/messages?peer_id=&page=&limit=` | 单聊历史（Bearer） |
| POST | `/api/groups` | 建群 `{name, member_ids}`（Bearer） |
| GET  | `/api/groups` | 我加入的群列表（Bearer） |
| POST | `/api/groups/members` | 拉人 `{group_id, member_ids}`（Bearer） |
| GET  | `/api/groups/messages?group_id=&page=&limit=` | 群历史（Bearer） |
| WS   | `/ws?token=<jwt>` | WebSocket 接入 |

### WebSocket 消息示例

```json
// 单聊
{"type":"chat","to":3,"content":"在吗"}

// 群聊
{"type":"group_chat","group_id":1,"content":"大家好"}

// 对方收到（群）
{"type":"group_chat","from":2,"from_name":"wym","group_id":1,"content":"大家好","ts":1735689600}
```

## 设计要点（面试可讲）

1. **Hub + channel**：`clients` map 只由 `Run()` 单 goroutine 写入；外部读用 RWMutex。
2. **读写分离**：每个连接 `readPump` / `writePump`；写连接只能走 `send` channel。
3. **非阻塞投递**：`SendToUser` 用 `select default`，慢客户端不会拖死发送方。
4. **离线补推**：单聊/群聊离线统一 Pending；上线后 `flushOffline` 按序补推。
5. **群聊写扩散**：遍历成员在线推 / 离线落库；历史只存一条主记录（`to_user_id=0`），避免重复。
6. **WS 鉴权放 query**：浏览器原生 WebSocket API 不支持自定义 Header。

## 目录结构

```
cmd/server/
internal/
  auth/ config/ db/ handler/ model/ store/ ws/
pkg/resp/
```

## 作者

wym
