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
		Preload("Assignee").
		Preload("Scopes").
		Preload("Milestones").
		Where("id = ?", id).First(&t).Error
	return t, err
}

func (d *TaskDao) ListByProject(projectID uint, page, limit int) ([]model.Task, int64, error) {
	var list []model.Task
	var total int64

	Database.Model(&model.Task{}).Where("project_id = ?", projectID).Count(&total)
	q := Database.Preload("Creator").Preload("Assignee").Preload("Scopes").Where("project_id = ?", projectID)
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *TaskDao) ListByProjectScoped(projectID uint, page, limit int, scope *UserScope) ([]model.Task, int64, error) {
	var list []model.Task
	var total int64

	q := Database.Model(&model.Task{}).Where("project_id = ?", projectID)
	q = scope.ApplyToTasks(q)
	q.Count(&total)

	q2 := Database.Preload("Creator").Preload("Assignee").Preload("Scopes").Where("project_id = ?", projectID)
	q2 = scope.ApplyToTasks(q2)
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *TaskDao) Update(t *model.Task) error {
	return Database.Save(t).Error
}

func (d *TaskDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Task{}, id).Error
}

func (d *TaskDao) RecalcCurrentValue(taskID uint) error {
	var task model.Task
	if err := Database.Select("aggregation_type", "start_value").Where("id = ?", taskID).First(&task).Error; err != nil {
		return err
	}

	aggType := task.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	switch aggType {
	case "SUM_DOWN":
		startVal := float64(0)
		if task.StartValue != nil {
			startVal = *task.StartValue
		}
		return Database.Exec(`
			UPDATE tasks SET current_value = ? - (
				SELECT COALESCE(SUM(achieved_value), 0)
				FROM milestones
				WHERE task_id = ? AND deleted_at IS NULL
			), updated_at = NOW()
			WHERE id = ?
		`, startVal, taskID, taskID).Error

	case "AVG":
		return Database.Exec(`
			UPDATE tasks SET current_value = (
				SELECT COALESCE(AVG(achieved_value), 0)
				FROM milestones
				WHERE task_id = ? AND deleted_at IS NULL
			), updated_at = NOW()
			WHERE id = ?
		`, taskID, taskID).Error

	default: // SUM_UP
		return Database.Exec(`
			UPDATE tasks SET current_value = (
				SELECT COALESCE(SUM(achieved_value), 0)
				FROM milestones
				WHERE task_id = ? AND deleted_at IS NULL
			), updated_at = NOW()
			WHERE id = ?
		`, taskID, taskID).Error
	}
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
		Preload("Assignee").
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Find(&list).Error
	return list, err
}

func (d *TaskDao) GetByProjectWithMilestones(projectID uint) ([]model.Task, error) {
	var list []model.Task
	err := Database.Preload("Milestones").Preload("Scopes").
		Preload("Assignee").
		Where("project_id = ?", projectID).Find(&list).Error
	return list, err
}

// GetByScopeEntity returns all tasks whose scope includes a specific ASC or Regiao.
func (d *TaskDao) GetByScopeEntity(scopeType string, scopeID uint) ([]model.Task, error) {
	var list []model.Task
	err := Database.
		Preload("Milestones").
		Preload("Assignee").
		Joins("JOIN task_scopes ON task_scopes.task_id = tasks.id").
		Where("task_scopes.scope_type = ? AND task_scopes.scope_id = ? AND tasks.deleted_at IS NULL", scopeType, scopeID).
		Find(&list).Error
	return list, err
}

// GetByScopeEntityInScope returns tasks that cover a specific ASC/Regiao AND are
// owned by the entities in the given UserScope (DirecaoIDs and/or DepartamentoIDs).
// Returns nil when the scope has no owner IDs (caller should fall back to global).
func (d *TaskDao) GetByScopeEntityInScope(scopeType string, scopeID uint, scope *UserScope) ([]model.Task, error) {
	hasDirs := len(scope.DirecaoIDs) > 0
	hasDepts := len(scope.DepartamentoIDs) > 0
	if !hasDirs && !hasDepts {
		return nil, nil
	}

	q := Database.
		Preload("Milestones").
		Preload("Assignee").
		Joins("JOIN task_scopes ON task_scopes.task_id = tasks.id").
		Where("task_scopes.scope_type = ? AND task_scopes.scope_id = ? AND tasks.deleted_at IS NULL",
			scopeType, scopeID)

	switch {
	case hasDirs && hasDepts:
		q = q.Where(
			Database.Where("tasks.owner_type = 'DIRECAO' AND tasks.owner_id IN ?", scope.DirecaoIDs).
				Or("tasks.owner_type = 'DEPARTAMENTO' AND tasks.owner_id IN ?", scope.DepartamentoIDs),
		)
	case hasDirs:
		q = q.Where("tasks.owner_type = 'DIRECAO' AND tasks.owner_id IN ?", scope.DirecaoIDs)
	default:
		q = q.Where("tasks.owner_type = 'DEPARTAMENTO' AND tasks.owner_id IN ?", scope.DepartamentoIDs)
	}

	var list []model.Task
	err := q.Find(&list).Error
	return list, err
}

// TaskSummary is a lightweight struct for scope stats.
type TaskSummary struct {
	ID           uint    `json:"id"`
	Title        string  `json:"title"`
	OwnerType    string  `json:"owner_type"`
	OwnerID      uint    `json:"owner_id"`
	ProjectID    uint    `json:"project_id"`
	ProjectTitle string  `json:"project_title"`
	CurrentValue float64 `json:"current_value"`
	TargetValue  float64 `json:"target_value"`
	Status       string  `json:"status"`
}

// SummaryByScopeEntity returns lightweight task rows scoped to a given entity.
func (d *TaskDao) SummaryByScopeEntity(scopeType string, scopeID uint) ([]TaskSummary, error) {
	var rows []TaskSummary
	err := Database.Raw(`
		SELECT t.id, t.title, t.owner_type, t.owner_id,
		       COALESCE(t.project_id, 0) AS project_id,
		       COALESCE(p.title, '') AS project_title,
		       t.current_value, t.target_value, t.status
		FROM tasks t
		JOIN task_scopes ts ON ts.task_id = t.id
		LEFT JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE ts.scope_type = ? AND ts.scope_id = ? AND t.deleted_at IS NULL
	`, scopeType, scopeID).Scan(&rows).Error
	return rows, err
}
