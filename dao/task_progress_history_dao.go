package dao

import (
	"kpi-backend/model"
	"time"
)

type TaskProgressHistoryDao struct{}

// upsertForPeriod saves or updates the task's value for the given YYYY-MM period.
func (d *TaskProgressHistoryDao) upsertForPeriod(taskID uint, period string, value float64, createdBy uint) error {
	var existing model.TaskProgressHistory
	err := Database.
		Where("task_id = ? AND period = ?", taskID, period).
		First(&existing).Error

	if err == nil {
		return Database.Model(&existing).
			Updates(map[string]interface{}{"value": value, "created_by": createdBy}).Error
	}
	record := model.TaskProgressHistory{
		TaskID:    taskID,
		Period:    period,
		Value:     value,
		CreatedBy: createdBy,
	}
	return Database.Create(&record).Error
}

// RecalcForPeriod recomputes the task's progress value for the month of the given
// milestone date, by aggregating all milestone achieved_values in that month.
// This is the main hook: called whenever a milestone is created/updated/deleted.
func (d *TaskProgressHistoryDao) RecalcForPeriod(taskID uint, milestoneDate time.Time, createdBy uint) error {
	period := milestoneDate.Format("2006-01")

	// Get task aggregation type and start_value
	var task model.Task
	if err := Database.Select("aggregation_type", "start_value").Where("id = ?", taskID).First(&task).Error; err != nil {
		return err
	}
	aggType := task.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	var value float64
	switch aggType {
	case "SUM_DOWN":
		startVal := 0.0
		if task.StartValue != nil {
			startVal = *task.StartValue
		}
		var sum float64
		Database.Raw(`
			SELECT COALESCE(SUM(achieved_value), 0)
			FROM milestones
			WHERE task_id = ? AND to_char(planned_date AT TIME ZONE 'UTC', 'YYYY-MM') = ? AND deleted_at IS NULL
		`, taskID, period).Scan(&sum)
		value = startVal - sum

	case "AVG":
		Database.Raw(`
			SELECT COALESCE(AVG(achieved_value), 0)
			FROM milestones
			WHERE task_id = ? AND to_char(planned_date AT TIME ZONE 'UTC', 'YYYY-MM') = ? AND deleted_at IS NULL AND achieved_value != 0
		`, taskID, period).Scan(&value)

	case "LAST":
		Database.Raw(`
			SELECT COALESCE(achieved_value, 0)
			FROM milestones
			WHERE task_id = ? AND to_char(planned_date AT TIME ZONE 'UTC', 'YYYY-MM') = ? AND deleted_at IS NULL AND achieved_value != 0
			ORDER BY updated_at DESC LIMIT 1
		`, taskID, period).Scan(&value)

	default: // SUM_UP
		Database.Raw(`
			SELECT COALESCE(SUM(achieved_value), 0)
			FROM milestones
			WHERE task_id = ? AND to_char(planned_date AT TIME ZONE 'UTC', 'YYYY-MM') = ? AND deleted_at IS NULL
		`, taskID, period).Scan(&value)
	}

	return d.upsertForPeriod(taskID, period, value, createdBy)
}

// UpsertProgress saves the task's current_value for today's month.
// Used only for MANUAL aggregation tasks (no milestone date to reference).
func (d *TaskProgressHistoryDao) UpsertProgress(taskID uint, value float64, createdBy uint) error {
	return d.upsertForPeriod(taskID, time.Now().Format("2006-01"), value, createdBy)
}

// UpsertProgressForPeriod saves the task's current_value for an explicit YYYY-MM period.
// Called from the PATCH /tasks/:id/progress endpoint when the user selects a period.
func (d *TaskProgressHistoryDao) UpsertProgressForPeriod(taskID uint, period string, value float64, createdBy uint) error {
	return d.upsertForPeriod(taskID, period, value, createdBy)
}

// ListByTaskPeriods returns the YYYY-MM period strings already recorded for a task.
// Used by the frontend to mark already-updated periods in the period selector.
func (d *TaskProgressHistoryDao) ListByTaskPeriods(taskID uint) ([]string, error) {
	var periods []string
	err := Database.
		Raw("SELECT period FROM task_progress_histories WHERE task_id = ? AND deleted_at IS NULL ORDER BY period ASC", taskID).
		Scan(&periods).Error
	return periods, err
}

// ListByTask returns all monthly progress records for a task, ordered by period.
func (d *TaskProgressHistoryDao) ListByTask(taskID uint) ([]model.TaskProgressHistory, error) {
	var list []model.TaskProgressHistory
	err := Database.
		Where("task_id = ? AND deleted_at IS NULL", taskID).
		Order("period ASC").
		Find(&list).Error
	return list, err
}
