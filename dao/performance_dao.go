package dao

import (
	"kpi-backend/model"
	"kpi-backend/util"
	"time"
)

type PerformanceDao struct{}

func (d *PerformanceDao) Upsert(cache *model.PerformanceCache) error {
	cache.ComputedAt = time.Now()
	cache.TrafficLight = util.GetTrafficLight(cache.TotalScore)

	var existing model.PerformanceCache
	err := Database.Where("entity_type = ? AND entity_id = ? AND period = ?",
		cache.EntityType, cache.EntityID, cache.Period).First(&existing).Error

	if err != nil {
		return Database.Create(cache).Error
	}

	return Database.Model(&existing).Updates(map[string]interface{}{
		"execution_score":    cache.ExecutionScore,
		"goal_score":         cache.GoalScore,
		"total_score":        cache.TotalScore,
		"traffic_light":      cache.TrafficLight,
		"tasks_total":        cache.TasksTotal,
		"tasks_completed":    cache.TasksCompleted,
		"milestones_total":   cache.MilestonesTotal,
		"milestones_done":    cache.MilestonesDone,
		"milestones_blocked": cache.MilestonesBlocked,
		"computed_at":        cache.ComputedAt,
	}).Error
}

func (d *PerformanceDao) GetScore(entityType string, entityID uint, period time.Time) (model.PerformanceCache, error) {
	var cache model.PerformanceCache
	err := Database.Where("entity_type = ? AND entity_id = ? AND period = ?",
		entityType, entityID, period).First(&cache).Error
	return cache, err
}

func (d *PerformanceDao) GetScores(entityType string, entityIDs []uint, period time.Time) ([]model.PerformanceCache, error) {
	var list []model.PerformanceCache
	err := Database.Where("entity_type = ? AND entity_id IN ? AND period = ?",
		entityType, entityIDs, period).Find(&list).Error
	return list, err
}

func (d *PerformanceDao) GetTimeline(entityType string, entityID uint, from, to time.Time) ([]model.PerformanceCache, error) {
	var list []model.PerformanceCache
	err := Database.Where("entity_type = ? AND entity_id = ? AND period >= ? AND period <= ?",
		entityType, entityID, from, to).Order("period ASC").Find(&list).Error
	return list, err
}

func (d *PerformanceDao) GetTopPerformers(entityType string, period time.Time, limit int) ([]model.PerformanceCache, error) {
	var list []model.PerformanceCache
	err := Database.Where("entity_type = ? AND period = ?", entityType, period).
		Order("total_score DESC").Limit(limit).Find(&list).Error
	return list, err
}

func (d *PerformanceDao) GetAllForPeriod(period time.Time) ([]model.PerformanceCache, error) {
	var list []model.PerformanceCache
	err := Database.Where("period = ?", period).Find(&list).Error
	return list, err
}

func (d *PerformanceDao) RefreshForTask(taskID uint) error {
	taskDao := TaskDao{}
	milestoneDao := MilestoneDao{}

	task, err := taskDao.GetByID(taskID)
	if err != nil {
		return err
	}

	milestones, err := milestoneDao.GetNonBlockedByTask(taskID)
	if err != nil {
		return err
	}

	var totalPlanned, totalAchieved float64
	for _, m := range milestones {
		totalPlanned += m.PlannedValue
		totalAchieved += m.AchievedValue
	}

	execScore := util.ComputeExecutionScore(totalPlanned, totalAchieved)

	startVal := float64(0)
	if task.StartValue != nil {
		startVal = *task.StartValue
	}
	goalScore := util.ComputeGoalScore(startVal, task.TargetValue, task.CurrentValue)
	totalScore := util.ComputePerformanceScore(execScore, goalScore)

	totalMs, doneMs, blockedMs := milestoneDao.CountByTask(taskID)

	now := time.Now()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	cache := &model.PerformanceCache{
		EntityType:        task.OwnerType,
		EntityID:          task.OwnerID,
		Period:            period,
		ExecutionScore:    execScore,
		GoalScore:         goalScore,
		TotalScore:        totalScore,
		TasksTotal:        1,
		TasksCompleted:    boolToInt(task.Status == "COMPLETED"),
		MilestonesTotal:   int(totalMs),
		MilestonesDone:    int(doneMs),
		MilestonesBlocked: int(blockedMs),
	}

	return d.Upsert(cache)
}

// PerformanceCacheAlias is used by dashboard controller for map lookups
type PerformanceCacheAlias = model.PerformanceCache

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
