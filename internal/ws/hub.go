// Package ws（hub.go）实现连接注册中心（经典 Hub 模式）
// 作者: wym
//
// 设计要点：
//   - clients map 只由 Run() 这一个 goroutine 写入/删除，避免并发写 map 崩溃
//   - 其他 goroutine 要读 clients（比如按 userID 查连接做单聊转发），走 RWMutex 保护的方法
//   - 在线状态额外写一份到 Redis Set，为后续多实例部署、"查询某用户是否在线"等场景做准备
//   - 用户上线时在 register 分支里补推离线消息，保证"先入 map 再推送"，避免 SendToUser 找不到连接
package ws

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/guyanxi11/im-server/internal/store"
)

// onlineSetKey 是 Redis 里记录在线用户 ID 集合的 key
const onlineSetKey = "im:online:users"

// Hub 是全局唯一的连接管理中心
type Hub struct {
	// register/unregister 是 Client 生命周期事件的输入 channel
	// 只有 Run() 里的 for-select 会消费它们，天然串行，不需要额外加锁
	register   chan *Client
	unregister chan *Client

	// clients 是 userID -> Client 的映射，代表当前进程内所有在线连接
	// mu 保护的是"外部 goroutine 读取 clients"这条路径（比如单聊转发时查目标用户连接）
	mu      sync.RWMutex
	clients map[uint]*Client

	rdb *redis.Client
	// msgStore 用于上线补推离线消息；单聊落库也走它
	msgStore *store.MessageStore
}

// NewHub 构造 Hub
func NewHub(rdb *redis.Client, msgStore *store.MessageStore) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint]*Client),
		rdb:        rdb,
		msgStore:   msgStore,
	}
}

// MsgStore 暴露给 Client 做消息落库，避免 Client 直接依赖全局变量
func (h *Hub) MsgStore() *store.MessageStore {
	return h.msgStore
}

// Run 是 Hub 的主循环，必须在 main 启动时用一个独立 goroutine 跑起来，且只跑一份
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			// 同一用户重复登录（多端/多标签页）场景：新连接顶替旧连接，
			// 避免旧的失效连接一直占着 clients map 和 Redis 在线标记
			if old, ok := h.clients[c.UserID]; ok {
				close(old.send)
			}
			h.clients[c.UserID] = c
			h.mu.Unlock()
			h.markOnline(c.UserID)
			log.Printf("[hub] user=%d online, total=%d", c.UserID, h.onlineCount())

			// 必须在写入 clients map 之后再补推，否则 SendToUser 会找不到连接
			// 异步执行：避免离线消息很多时卡住 Hub 主循环处理其他上下线事件
			go h.flushOffline(c.UserID)

		case c := <-h.unregister:
			h.mu.Lock()
			// 只有当 map 里的连接确实还是这个 c 时才删除/关闭，
			// 防止"旧连接的 unregister 事件"误删了"新连接"（配合上面顶替逻辑）
			if cur, ok := h.clients[c.UserID]; ok && cur == c {
				delete(h.clients, c.UserID)
				close(c.send)
			}
			h.mu.Unlock()
			h.markOffline(c.UserID)
			log.Printf("[hub] user=%d offline, total=%d", c.UserID, h.onlineCount())
		}
	}
}

// flushOffline 把该用户所有 Pending 消息按时间顺序推过去，成功后标记 Delivered
func (h *Hub) flushOffline(userID uint) {
	list, err := h.msgStore.ListPendingByToUser(userID)
	if err != nil {
		log.Printf("[hub] list pending failed, user=%d err=%v", userID, err)
		return
	}
	if len(list) == 0 {
		return
	}

	deliveredIDs := make([]uint, 0, len(list))
	for _, m := range list {
		payload, err := encodeOutbound(OutboundMessage{
			Type:     MsgTypeChat,
			From:     m.FromUserID,
			FromName: m.FromUsername,
			To:       m.ToUserID,
			Content:  m.Content,
			TS:       m.CreatedAt.Unix(),
		})
		if err != nil {
			continue
		}
		if h.SendToUser(userID, payload) {
			deliveredIDs = append(deliveredIDs, m.ID)
		}
	}

	if err := h.msgStore.MarkDelivered(deliveredIDs); err != nil {
		log.Printf("[hub] mark delivered failed, user=%d err=%v", userID, err)
		return
	}
	log.Printf("[hub] flushed %d offline msgs to user=%d", len(deliveredIDs), userID)
}

// GetClient 按 userID 查找当前进程内的连接，找不到返回 nil, false
func (h *Hub) GetClient(userID uint) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[userID]
	return c, ok
}

// SendToUser 向指定用户推送一条下行消息
// 返回 true 表示已成功塞入目标连接的 send channel；false 表示用户不在线或发送缓冲已满
// 注意：这里用 non-blocking send（select default），避免目标用户卡死导致发送方 goroutine 阻塞泄漏
func (h *Hub) SendToUser(userID uint, payload []byte) bool {
	h.mu.RLock()
	c, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return false
	}

	select {
	case c.send <- payload:
		return true
	default:
		// send 缓冲已满：说明对端消费太慢，丢弃本条并记日志
		log.Printf("[hub] send buffer full, drop msg to user=%d", userID)
		return false
	}
}

// onlineCount 返回当前进程内在线连接数，仅用于日志观察
func (h *Hub) onlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// markOnline 把用户 ID 写入 Redis 在线集合
func (h *Hub) markOnline(userID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.rdb.SAdd(ctx, onlineSetKey, strconv.FormatUint(uint64(userID), 10)).Err(); err != nil {
		log.Printf("[hub] mark online failed, user=%d err=%v", userID, err)
	}
}

// markOffline 把用户 ID 从 Redis 在线集合移除
func (h *Hub) markOffline(userID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.rdb.SRem(ctx, onlineSetKey, strconv.FormatUint(uint64(userID), 10)).Err(); err != nil {
		log.Printf("[hub] mark offline failed, user=%d err=%v", userID, err)
	}
}

// OnlineUserIDs 从 Redis 读取当前所有在线用户 ID，供 HTTP 接口查询展示
func (h *Hub) OnlineUserIDs(ctx context.Context) ([]string, error) {
	return h.rdb.SMembers(ctx, onlineSetKey).Result()
}
