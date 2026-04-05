package dao

import (
	"kpi-backend/model"
	"time"
)

type NotificationDao struct{}

func (d *NotificationDao) Create(n *model.Notification) error {
	n.CreatedAt = time.Now()
	return Database.Create(n).Error
}

func (d *NotificationDao) ListByUser(userID uint, page, limit int, isRead *bool) ([]model.Notification, int64, error) {
	var list []model.Notification
	var total int64

	q := Database.Model(&model.Notification{}).Where("user_id = ?", userID)
	if isRead != nil {
		q = q.Where("is_read = ?", *isRead)
	}
	q.Count(&total)

	q2 := Database.Where("user_id = ?", userID)
	if isRead != nil {
		q2 = q2.Where("is_read = ?", *isRead)
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *NotificationDao) MarkRead(id, userID uint) error {
	return Database.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (d *NotificationDao) MarkAllRead(userID uint) error {
	return Database.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (d *NotificationDao) CreateAndEmail(userID uint, title, message, notifType, entityType string, entityID *uint) {
	n := &model.Notification{
		UserID:     userID,
		Title:      title,
		Message:    message,
		Type:       notifType,
		EntityType: entityType,
		EntityID:   entityID,
	}
	d.Create(n)
}

func (d *NotificationDao) UnreadCount(userID uint) int64 {
	var count int64
	Database.Model(&model.Notification{}).Where("user_id = ? AND is_read = false", userID).Count(&count)
	return count
}
