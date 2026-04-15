package dao

import (
	"kpi-backend/model"

	"gorm.io/gorm"
)

type FeedbackDao struct{}

func (d *FeedbackDao) Create(f *model.Feedback) error {
	return Database.Create(f).Error
}

func (d *FeedbackDao) GetByID(id uint) (model.Feedback, error) {
	var f model.Feedback
	err := Database.
		Preload("Sender").
		Preload("Receiver").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Sender").Order("created_at ASC")
		}).
		Where("id = ?", id).First(&f).Error
	return f, err
}

func (d *FeedbackDao) ListReceived(userID uint, page, limit int) ([]model.Feedback, int64, error) {
	var list []model.Feedback
	var total int64

	q := Database.Model(&model.Feedback{}).
		Where("receiver_id = ? AND parent_id IS NULL", userID)

	q.Count(&total)

	err := q.
		Preload("Sender").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Sender").Order("created_at ASC")
		}).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&list).Error

	return list, total, err
}

func (d *FeedbackDao) ListSent(userID uint, page, limit int) ([]model.Feedback, int64, error) {
	var list []model.Feedback
	var total int64

	q := Database.Model(&model.Feedback{}).
		Where("sender_id = ? AND parent_id IS NULL", userID)

	q.Count(&total)

	err := q.
		Preload("Receiver").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&list).Error

	return list, total, err
}

func (d *FeedbackDao) ListForDashboard(userID uint, limit int) ([]model.Feedback, error) {
	var list []model.Feedback
	err := Database.
		Preload("Sender").
		Where("receiver_id = ? AND is_read = false AND parent_id IS NULL", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (d *FeedbackDao) MarkAsRead(id uint) error {
	return Database.Model(&model.Feedback{}).Where("id = ?", id).Update("is_read", true).Error
}

func (d *FeedbackDao) CountUnread(userID uint) (int64, error) {
	var count int64
	err := Database.Model(&model.Feedback{}).
		Where("receiver_id = ? AND is_read = false AND parent_id IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (d *FeedbackDao) ListByTarget(targetType string, targetID uint, page, limit int) ([]model.Feedback, int64, error) {
	var list []model.Feedback
	var total int64

	q := Database.Model(&model.Feedback{}).
		Where("target_type = ? AND target_id = ? AND parent_id IS NULL", targetType, targetID)

	q.Count(&total)

	err := q.
		Preload("Sender").
		Preload("Receiver").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Sender").Order("created_at ASC")
		}).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&list).Error

	return list, total, err
}

func (d *FeedbackDao) Delete(id uint) error {
	return Database.Delete(&model.Feedback{}, id).Error
}
