package dao

import (
	"kpi-backend/model"
	"time"
)

type MilestoneDao struct{}

func (d *MilestoneDao) Create(m *model.Milestone) error {
	return Database.Create(m).Error
}

func (d *MilestoneDao) GetByID(id uint) (model.Milestone, error) {
	var m model.Milestone
	err := Database.Preload("Creator").Preload("Updater").Preload("Assignee").Preload("Task").Where("id = ?", id).First(&m).Error
	return m, err
}

func (d *MilestoneDao) ListByTask(taskID uint, page, limit int) ([]model.Milestone, int64, error) {
	var list []model.Milestone
	var total int64

	Database.Model(&model.Milestone{}).Where("task_id = ?", taskID).Count(&total)
	q := Database.Preload("Creator").Preload("Assignee").Where("task_id = ?", taskID)
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("planned_date ASC").Find(&list).Error
	return list, total, err
}

func (d *MilestoneDao) Update(m *model.Milestone) error {
	return Database.Save(m).Error
}

func (d *MilestoneDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Milestone{}, id).Error
}

func (d *MilestoneDao) UpdatePhoto(id uint, photoURL string) error {
	return Database.Model(&model.Milestone{}).Where("id = ?", id).Update("photo_url", photoURL).Error
}

func (d *MilestoneDao) GetOverdue() ([]model.Milestone, error) {
	var list []model.Milestone
	err := Database.Preload("Task").
		Where("status = 'PENDING' AND planned_date < ? AND deleted_at IS NULL", time.Now()).
		Find(&list).Error
	return list, err
}

func (d *MilestoneDao) GetNonBlockedByTask(taskID uint) ([]model.Milestone, error) {
	var list []model.Milestone
	err := Database.Where("task_id = ? AND status != 'BLOCKED' AND deleted_at IS NULL", taskID).Find(&list).Error
	return list, err
}

func (d *MilestoneDao) CountByTask(taskID uint) (total, done, blocked int64) {
	Database.Model(&model.Milestone{}).Where("task_id = ? AND deleted_at IS NULL", taskID).Count(&total)
	Database.Model(&model.Milestone{}).Where("task_id = ? AND status = 'DONE' AND deleted_at IS NULL", taskID).Count(&done)
	Database.Model(&model.Milestone{}).Where("task_id = ? AND status = 'BLOCKED' AND deleted_at IS NULL", taskID).Count(&blocked)
	return
}
