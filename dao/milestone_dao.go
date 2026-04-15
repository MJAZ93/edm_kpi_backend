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

// RecalcAchievedValue recalculates a milestone's achieved_value from its progress entries
// based on the milestone's aggregation_type. If the milestone has no aggregation_type set,
// it inherits from the parent task's aggregation_type.
func (d *MilestoneDao) RecalcAchievedValue(milestoneID uint) error {
	var ms model.Milestone
	if err := Database.Select("aggregation_type", "task_id").Where("id = ?", milestoneID).First(&ms).Error; err != nil {
		return err
	}

	aggType := ms.AggregationType
	if aggType == "" {
		// Inherit from parent task
		var task model.Task
		if err := Database.Select("aggregation_type").Where("id = ?", ms.TaskID).First(&task).Error; err == nil && task.AggregationType != "" {
			aggType = task.AggregationType
		} else {
			aggType = "SUM_UP"
		}
	}

	switch aggType {
	case "AVG":
		return Database.Exec(`
			UPDATE milestones SET achieved_value = (
				SELECT COALESCE(AVG(increment_value), 0)
				FROM milestone_progresses
				WHERE milestone_id = ? AND deleted_at IS NULL
			), updated_at = NOW()
			WHERE id = ?
		`, milestoneID, milestoneID).Error

	case "LAST":
		return Database.Exec(`
			UPDATE milestones SET achieved_value = COALESCE((
				SELECT increment_value
				FROM milestone_progresses
				WHERE milestone_id = ? AND deleted_at IS NULL
				ORDER BY updated_at DESC
				LIMIT 1
			), 0), updated_at = NOW()
			WHERE id = ?
		`, milestoneID, milestoneID).Error

	default: // SUM_UP
		return Database.Exec(`
			UPDATE milestones SET achieved_value = (
				SELECT COALESCE(SUM(increment_value), 0)
				FROM milestone_progresses
				WHERE milestone_id = ? AND deleted_at IS NULL
			), updated_at = NOW()
			WHERE id = ?
		`, milestoneID, milestoneID).Error
	}
}

// UpdateProgress updates a single progress entry's increment_value and notes.
func (d *MilestoneDao) UpdateProgress(progressID uint, incrementValue float64, notes string) error {
	return Database.Model(&model.MilestoneProgress{}).Where("id = ?", progressID).
		Updates(map[string]interface{}{
			"increment_value": incrementValue,
			"notes":           notes,
		}).Error
}
