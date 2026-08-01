// Package ws（client.go）定义单个 WebSocket 连接的读写逻辑
// 作者: wym
//
// 关键约束：gorilla/websocket 的 *websocket.Conn 不允许多个 goroutine 并发写。
// 所以每个连接固定用两个 goroutine：
//   - readPump：唯一的读者，负责从连接里读消息
//   - writePump：唯一的写者，从 send channel 里取消息写回连接
//
// 其他 goroutine（比如处理单聊转发的逻辑）要给这个连接发消息，
// 只能往 Client.send 这个 channel 塞数据，不能直接调用 conn.WriteMessage。
package ws

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/wym/im-server/internal/model"
)

const (
	// writeWait 是单次写操作的超时时间
	writeWait = 10 * time.Second
	// pongWait 是等待客户端 pong 响应的超时时间，超时视为连接已死
	pongWait = 60 * time.Second
	// pingPeriod 必须小于 pongWait，服务端按此周期主动发 ping
	pingPeriod = (pongWait * 9) / 10
	// maxMessageSize 限制单条消息大小，防止恶意超大消息占满内存
	maxMessageSize = 4096
)

// Client 代表一个已认证的 WebSocket 连接
type Client struct {
	hub  *Hub
	conn *websocket.Conn

	// UserID/Username 来自 JWT 解析结果，代表这条连接背后的用户身份
	UserID   uint
	Username string

	// send 是这个连接的出站消息缓冲区，writePump 是唯一消费者
	// 带缓冲是为了应对瞬时消息突发，避免发送方阻塞
	send chan []byte
}

// readPump 循环读取客户端发来的消息，解析后按类型分发
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// 收到客户端的 pong 后刷新读超时，只要客户端还在正常心跳，连接就不会被判定超时
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("[ws] user=%d read closed: %v", c.UserID, err)
			return
		}
		c.handleInbound(payload)
	}
}

// handleInbound 解析上行 JSON 并按 type 分发处理
func (c *Client) handleInbound(payload []byte) {
	var in InboundMessage
	if err := json.Unmarshal(payload, &in); err != nil {
		c.replyError("消息格式错误，需要 JSON")
		return
	}

	switch in.Type {
	case MsgTypePing:
		c.replyPong()
	case MsgTypeChat:
		c.handleChat(in)
	default:
		c.replyError("未知消息类型: " + in.Type)
	}
}

// handleChat 处理单聊：校验 -> 尝试在线投递 -> 落库（在线 Delivered / 离线 Pending）-> 回执
func (c *Client) handleChat(in InboundMessage) {
	content := strings.TrimSpace(in.Content)
	if in.To == 0 {
		c.replyError("缺少目标用户 to")
		return
	}
	if content == "" {
		c.replyError("消息内容不能为空")
		return
	}
	// 禁止自己给自己发（业务上无意义；若要支持"笔记"类场景可去掉此限制）
	if in.To == c.UserID {
		c.replyError("不能给自己发消息")
		return
	}

	out := OutboundMessage{
		Type:     MsgTypeChat,
		From:     c.UserID,
		FromName: c.Username,
		To:       in.To,
		Content:  content,
		TS:       time.Now().Unix(),
	}
	payload, err := encodeOutbound(out)
	if err != nil {
		c.replyError("消息序列化失败")
		return
	}

	// 先尝试在线投递，再按结果决定落库状态
	// 顺序说明：如果先落库再推送，推送成功但落库失败会导致"对方看到了但库里没有"；
	// 当前选择"先推后存"：极端情况下库失败只影响历史记录，不影响实时体验。
	// 生产级 IM 通常用"先写库再推 + 本地队列重试"，这里按练手项目简化。
	delivered := c.hub.SendToUser(in.To, payload)
	status := model.MsgStatusPending
	ackMsg := "saved_offline"
	if delivered {
		status = model.MsgStatusDelivered
		ackMsg = "delivered"
	}

	if _, err := c.hub.MsgStore().SaveChat(c.UserID, c.Username, in.To, content, status); err != nil {
		log.Printf("[ws] save chat failed: %v", err)
		c.replyError("消息保存失败")
		return
	}

	c.replyAck(true, ackMsg)
	if delivered {
		log.Printf("[ws] chat %d -> %d delivered", c.UserID, in.To)
	} else {
		log.Printf("[ws] chat %d -> %d saved offline", c.UserID, in.To)
	}
}

// replyError 向本连接推送一条错误通知（走 send channel，不直接写 conn）
func (c *Client) replyError(msg string) {
	payload, err := encodeOutbound(OutboundMessage{Type: MsgTypeError, Msg: msg})
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
		log.Printf("[ws] user=%d send buffer full, drop error reply", c.UserID)
	}
}

// replyAck 向本连接推送投递回执
func (c *Client) replyAck(ok bool, msg string) {
	payload, err := encodeOutbound(OutboundMessage{Type: MsgTypeAck, OK: ok, Msg: msg})
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
		log.Printf("[ws] user=%d send buffer full, drop ack", c.UserID)
	}
}

// replyPong 响应应用层心跳
func (c *Client) replyPong() {
	payload, err := encodeOutbound(OutboundMessage{Type: MsgTypePong})
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
	}
}

// writePump 是这个连接唯一允许调用 conn.WriteMessage 的地方
// 同时承担定时发送 ping 的职责，双重保证：应用层心跳 + WebSocket 协议层 ping/pong
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub 关闭了 send channel，说明这个连接已被注销，通知对端后退出
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("[ws] user=%d write failed: %v", c.UserID, err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
