// Package store 封装数据库读写，把 SQL 细节从 ws/handler 里抽出来
// 作者: wym
package store

import (
	"time"

	"gorm.io/gorm"

	"github.com/guyanxi11/im-server/internal/model"
)

// MessageStore 是消息表的数据访问层
type MessageStore struct {
	db *gorm.DB
}

// NewMessageStore 构造 MessageStore
func NewMessageStore(db *gorm.DB) *MessageStore {
	return &MessageStore{db: db}
}

// SaveChat 持久化一条单聊消息，返回带自增 ID 的记录
// status 由调用方决定：在线投递成功写 Delivered，否则写 Pending
func (s *MessageStore) SaveChat(fromID uint, fromName string, toID uint, content string, status int) (*model.Message, error) {
	msg := &model.Message{
		FromUserID:   fromID,
		FromUsername: fromName,
		ToUserID:     toID,
		Content:      content,
		Status:       status,
	}
	if status == model.MsgStatusDelivered {
		now := time.Now()
		msg.DeliveredAt = &now
	}
	if err := s.db.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

// ListPendingByToUser 按创建时间升序取出某用户所有待投递（离线）消息
// 升序很重要：补推时要保证对方看到的顺序和发送时间一致
func (s *MessageStore) ListPendingByToUser(toUserID uint) ([]model.Message, error) {
	var list []model.Message
	err := s.db.Where("to_user_id = ? AND status = ?", toUserID, model.MsgStatusPending).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// MarkDelivered 把指定消息批量标记为已投递
func (s *MessageStore) MarkDelivered(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return s.db.Model(&model.Message{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":       model.MsgStatusDelivered,
			"delivered_at": now,
		}).Error
}

// ListHistory 分页查询两人之间的单聊历史
// 返回：消息列表（按 id 降序，最新在前）、总数
// 查询条件：(A->B) OR (B->A)，保证双向会话都能查到
func (s *MessageStore) ListHistory(userID, peerID uint, page, limit int) ([]model.Message, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // 防止一次拉太多拖垮接口
	}

	q := s.db.Model(&model.Message{}).Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		userID, peerID, peerID, userID,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Message
	offset := (page - 1) * limit
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}
