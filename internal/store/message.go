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
	return s.saveMsg(fromID, fromName, toID, 0, content, status)
}

// SaveGroupHistory 写入一条群聊"会话主记录"（ToUserID=0），供群历史分页查询
func (s *MessageStore) SaveGroupHistory(fromID uint, fromName string, groupID uint, content string) (*model.Message, error) {
	return s.saveMsg(fromID, fromName, 0, groupID, content, model.MsgStatusDelivered)
}

// SaveGroupOffline 为某个离线群成员写入待补推副本
func (s *MessageStore) SaveGroupOffline(fromID uint, fromName string, toID, groupID uint, content string) (*model.Message, error) {
	return s.saveMsg(fromID, fromName, toID, groupID, content, model.MsgStatusPending)
}

func (s *MessageStore) saveMsg(fromID uint, fromName string, toID, groupID uint, content string, status int) (*model.Message, error) {
	msg := &model.Message{
		FromUserID:   fromID,
		FromUsername: fromName,
		ToUserID:     toID,
		GroupID:      groupID,
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

// ListHistory 分页查询两人之间的单聊历史（排除群聊记录）
func (s *MessageStore) ListHistory(userID, peerID uint, page, limit int) ([]model.Message, int64, error) {
	page, limit = normalizePage(page, limit)

	q := s.db.Model(&model.Message{}).Where(
		"group_id = 0 AND ((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
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

// ListGroupHistory 分页查询群聊历史（只查 ToUserID=0 的会话主记录，避免离线副本重复）
func (s *MessageStore) ListGroupHistory(groupID uint, page, limit int) ([]model.Message, int64, error) {
	page, limit = normalizePage(page, limit)

	q := s.db.Model(&model.Message{}).Where("group_id = ? AND to_user_id = 0", groupID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Message
	offset := (page - 1) * limit
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
