// Package store 封装数据库读写，�?SQL 细节�?ws/handler 里抽出来
// 作�? wym
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

// NewMessageStore 构�?MessageStore
func NewMessageStore(db *gorm.DB) *MessageStore {
	return &MessageStore{db: db}
}

// SaveChat 持久化一条单聊消息，返回带自�?ID 的记�?// status 由调用方决定：在线投递成功写 Delivered，否则写 Pending
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

// ListPendingByToUser 按创建时间升序取出某用户所有待投递（离线）消�?// 升序很重要：补推时要保证对方看到的顺序和发送时间一�?func (s *MessageStore) ListPendingByToUser(toUserID uint) ([]model.Message, error) {
	var list []model.Message
	err := s.db.Where("to_user_id = ? AND status = ?", toUserID, model.MsgStatusPending).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// MarkDelivered 把指定消息批量标记为已投�?func (s *MessageStore) MarkDelivered(ids []uint) error {
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
