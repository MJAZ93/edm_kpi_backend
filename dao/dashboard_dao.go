package dao

import (
	"kpi-backend/model"
)

type DashboardDao struct{}

type DashboardSummary struct {
	TotalProjects     int64   `json:"total_projects"`
	TotalTasks        int64   `json:"total_tasks"`
	MilestonesDone    int64   `json:"milestones_done"`
	MilestonesPending int64   `json:"milestones_pending"`
	MilestonesBlocked int64   `json:"milestones_blocked"`
	ExecutionScore    float64 `json:"execution_score"`
	GoalScore         float64 `json:"goal_score"`
	TotalScore        float64 `json:"total_score"`
	TrafficLight      string  `json:"traffic_light"`
}

func (d *DashboardDao) GetSummary() DashboardSummary {
	var s DashboardSummary

	Database.Model(&model.Project{}).Where("status = 'ACTIVE'").Count(&s.TotalProjects)
	Database.Model(&model.Task{}).Where("status = 'ACTIVE'").Count(&s.TotalTasks)
	Database.Model(&model.Milestone{}).Where("status = 'DONE' AND deleted_at IS NULL").Count(&s.MilestonesDone)
	Database.Model(&model.Milestone{}).Where("status = 'PENDING' AND deleted_at IS NULL").Count(&s.MilestonesPending)
	Database.Model(&model.Milestone{}).Where("status = 'BLOCKED' AND deleted_at IS NULL").Count(&s.MilestonesBlocked)

	var totalPlanned, totalAchieved float64
	Database.Model(&model.Milestone{}).
		Where("status != 'BLOCKED' AND deleted_at IS NULL").
		Select("COALESCE(SUM(planned_value), 0)").Scan(&totalPlanned)
	Database.Model(&model.Milestone{}).
		Where("status != 'BLOCKED' AND deleted_at IS NULL").
		Select("COALESCE(SUM(achieved_value), 0)").Scan(&totalAchieved)

	if totalPlanned > 0 {
		s.ExecutionScore = (totalAchieved / totalPlanned) * 100
	}

	var tasks []model.Task
	Database.Where("status = 'ACTIVE' AND target_value > 0").Find(&tasks)
	if len(tasks) > 0 {
		var sumGoal float64
		for _, t := range tasks {
			startVal := float64(0)
			if t.StartValue != nil {
				startVal = *t.StartValue
			}
			diff := t.TargetValue - startVal
			if diff > 0 {
				sumGoal += ((t.CurrentValue - startVal) / diff) * 100
			}
		}
		s.GoalScore = sumGoal / float64(len(tasks))
	}

	s.TotalScore = (s.ExecutionScore * 0.6) + (s.GoalScore * 0.4)

	if s.TotalScore >= 90 {
		s.TrafficLight = "GREEN"
	} else if s.TotalScore >= 60 {
		s.TrafficLight = "YELLOW"
	} else {
		s.TrafficLight = "RED"
	}

	return s
}

func (d *DashboardDao) GetSummaryScoped(scope *UserScope) DashboardSummary {
	if scope.IsGlobal {
		return d.GetSummary()
	}

	var s DashboardSummary

	// Scoped project count
	pq := Database.Model(&model.Project{}).Where("status = 'ACTIVE'")
	pq = scope.ApplyToProjects(pq)
	pq.Count(&s.TotalProjects)

	// Scoped task count + task IDs for milestone filtering
	tq := Database.Model(&model.Task{}).Where("status = 'ACTIVE'")
	tq = scope.ApplyToTasks(tq)
	tq.Count(&s.TotalTasks)

	taskIDs := scope.TaskIDsSubquery()

	// Scoped milestone counts
	Database.Model(&model.Milestone{}).
		Where("status = 'DONE' AND deleted_at IS NULL AND task_id IN (?)", taskIDs).
		Count(&s.MilestonesDone)
	Database.Model(&model.Milestone{}).
		Where("status = 'PENDING' AND deleted_at IS NULL AND task_id IN (?)", taskIDs).
		Count(&s.MilestonesPending)
	Database.Model(&model.Milestone{}).
		Where("status = 'BLOCKED' AND deleted_at IS NULL AND task_id IN (?)", taskIDs).
		Count(&s.MilestonesBlocked)

	// Scoped execution score
	var totalPlanned, totalAchieved float64
	Database.Model(&model.Milestone{}).
		Where("status != 'BLOCKED' AND deleted_at IS NULL AND task_id IN (?)", taskIDs).
		Select("COALESCE(SUM(planned_value), 0)").Scan(&totalPlanned)
	Database.Model(&model.Milestone{}).
		Where("status != 'BLOCKED' AND deleted_at IS NULL AND task_id IN (?)", taskIDs).
		Select("COALESCE(SUM(achieved_value), 0)").Scan(&totalAchieved)

	if totalPlanned > 0 {
		s.ExecutionScore = (totalAchieved / totalPlanned) * 100
	}

	// Scoped goal score
	var tasks []model.Task
	tq2 := Database.Where("status = 'ACTIVE' AND target_value > 0")
	tq2 = scope.ApplyToTasks(tq2)
	tq2.Find(&tasks)
	if len(tasks) > 0 {
		var sumGoal float64
		for _, t := range tasks {
			startVal := float64(0)
			if t.StartValue != nil {
				startVal = *t.StartValue
			}
			diff := t.TargetValue - startVal
			if diff > 0 {
				sumGoal += ((t.CurrentValue - startVal) / diff) * 100
			}
		}
		s.GoalScore = sumGoal / float64(len(tasks))
	}

	s.TotalScore = (s.ExecutionScore * 0.6) + (s.GoalScore * 0.4)

	if s.TotalScore >= 90 {
		s.TrafficLight = "GREEN"
	} else if s.TotalScore >= 60 {
		s.TrafficLight = "YELLOW"
	} else {
		s.TrafficLight = "RED"
	}

	return s
}

type DistributionItem struct {
	Label      string  `json:"label"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (d *DashboardDao) GetDistribution(dimension string) []DistributionItem {
	var items []DistributionItem

	switch dimension {
	case "BY_STATUS":
		d.countByField("tasks", "status", &items)
	case "BY_TRAFFIC_LIGHT":
		d.countByField("performance_caches", "traffic_light", &items)
	case "BY_OWNER_TYPE":
		d.countByField("tasks", "owner_type", &items)
	}

	var total int64
	for _, i := range items {
		total += i.Count
	}
	for idx := range items {
		if total > 0 {
			items[idx].Percentage = float64(items[idx].Count) / float64(total) * 100
		}
	}

	return items
}

func (d *DashboardDao) countByField(table, field string, items *[]DistributionItem) {
	Database.Table(table).
		Select(field + " as label, COUNT(*) as count").
		Group(field).
		Order("count DESC").
		Scan(items)
}

type BenchmarkResult struct {
	A       BenchmarkEntity `json:"a"`
	B       BenchmarkEntity `json:"b"`
	Ratio   float64         `json:"ratio"`
	Message string          `json:"message"`
}

type BenchmarkEntity struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	TotalScore float64 `json:"total_score"`
}
