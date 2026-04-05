package dao

import "kpi-backend/model"

type TaskDao struct{}

func (d *TaskDao) Create(t *model.Task) error {
	return Database.Create(t).Error
}

func (d *TaskDao) GetByID(id uint) (model.Task, error) {
	var t model.Task
	err := Database.
		Preload("Creator").
		Preload("Scopes").
		Preload("Milestones").
		Where("id = ?", id).First(&t).Error
	return t, err
}

func (d *TaskDao) ListByProject(projectID uint, page, limit int) ([]model.Task, int64, error) {
	var list []model.Task
	var total int64

	Database.Model(&model.Task{}).Where("project_id = ?", projectID).Count(&total)
	q := Database.Preload("Creator").Preload("Scopes").Where("project_id = ?", projectID)
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *TaskDao) Update(t *model.Task) error {
	return Database.Save(t).Error
}

func (d *TaskDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Task{}, id).Error
}

func (d *TaskDao) RecalcCurrentValue(taskID uint) error {
	return Database.Exec(`
		UPDATE tasks SET current_value = (
			SELECT COALESCE(SUM(achieved_value), 0)
			FROM milestones
			WHERE task_id = ? AND status != 'BLOCKED' AND deleted_at IS NULL
		), updated_at = NOW()
		WHERE id = ?
	`, taskID, taskID).Error
}

func (d *TaskDao) CreateScopes(scopes []model.TaskScope) error {
	if len(scopes) == 0 {
		return nil
	}
	return Database.Create(&scopes).Error
}

func (d *TaskDao) DeleteScopes(taskID uint) error {
	return Database.Where("task_id = ?", taskID).Delete(&model.TaskScope{}).Error
}

func (d *TaskDao) ListActive() ([]model.Task, error) {
	var list []model.Task
	err := Database.Where("status = 'ACTIVE' AND end_date IS NOT NULL").Find(&list).Error
	return list, err
}

func (d *TaskDao) ListByOwner(ownerType string, ownerID uint) ([]model.Task, error) {
	var list []model.Task
	err := Database.Preload("Scopes").Preload("Milestones").
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Find(&list).Error
	return list, err
}

func (d *TaskDao) GetByProjectWithMilestones(projectID uint) ([]model.Task, error) {
	var list []model.Task
	err := Database.Preload("Milestones").Preload("Scopes").
		Where("project_id = ?", projectID).Find(&list).Error
	return list, err
}
