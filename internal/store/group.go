// Package store（group.go）封装群与群成员的数据访问
// 作者: wym
package store

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/guyanxi11/im-server/internal/model"
)

// ErrNotGroupMember 表示当前用户不是该群成员
var ErrNotGroupMember = errors.New("not a group member")

// ErrGroupNotFound 表示群不存在
var ErrGroupNotFound = errors.New("group not found")

// GroupStore 是群相关表的数据访问层
type GroupStore struct {
	db *gorm.DB
}

// NewGroupStore 构造 GroupStore
func NewGroupStore(db *gorm.DB) *GroupStore {
	return &GroupStore{db: db}
}

// CreateGroup 创建群并把 owner + 初始成员写入 group_members（事务）
// memberIDs 可不包含 owner，方法内会自动确保 owner 在成员列表中
func (s *GroupStore) CreateGroup(ownerID uint, name string, memberIDs []uint) (*model.Group, error) {
	g := &model.Group{
		Name:    name,
		OwnerID: ownerID,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(g).Error; err != nil {
			return err
		}

		// 用 map 去重，并保证群主一定在成员里
		uniq := map[uint]struct{}{ownerID: {}}
		for _, id := range memberIDs {
			if id == 0 {
				continue
			}
			uniq[id] = struct{}{}
		}

		now := time.Now()
		members := make([]model.GroupMember, 0, len(uniq))
		for uid := range uniq {
			members = append(members, model.GroupMember{
				GroupID:  g.ID,
				UserID:   uid,
				JoinedAt: now,
			})
		}
		return tx.Create(&members).Error
	})
	if err != nil {
		return nil, err
	}
	return g, nil
}

// AddMembers 向已有群添加成员；调用方必须已是群成员（通常是群主，这里放宽为任意成员可拉人，练手简化）
func (s *GroupStore) AddMembers(groupID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	now := time.Now()
	members := make([]model.GroupMember, 0, len(userIDs))
	seen := map[uint]struct{}{}
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		members = append(members, model.GroupMember{
			GroupID:  groupID,
			UserID:   uid,
			JoinedAt: now,
		})
	}
	// 忽略重复加入错误：用 FirstOrCreate 更稳，但批量时用 Clauses OnConflict 依赖方言；
	// 这里逐条处理，已存在则跳过
	for _, m := range members {
		var existing model.GroupMember
		err := s.db.Where("group_id = ? AND user_id = ?", m.GroupID, m.UserID).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := s.db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

// IsMember 判断用户是否在群内
func (s *GroupStore) IsMember(groupID, userID uint) (bool, error) {
	var count int64
	err := s.db.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error
	return count > 0, err
}

// ListMemberIDs 返回群内所有成员 userID
func (s *GroupStore) ListMemberIDs(groupID uint) ([]uint, error) {
	var ids []uint
	err := s.db.Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// GetGroup 按 ID 查群
func (s *GroupStore) GetGroup(groupID uint) (*model.Group, error) {
	var g model.Group
	err := s.db.First(&g, groupID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGroupNotFound
	}
	return &g, err
}

// ListGroupsByUser 列出用户加入的所有群
func (s *GroupStore) ListGroupsByUser(userID uint) ([]model.Group, error) {
	var groups []model.Group
	err := s.db.Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Order("groups.id DESC").
		Find(&groups).Error
	return groups, err
}
