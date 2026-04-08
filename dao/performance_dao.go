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

// RefreshForProject recomputes execution+goal scores for a project from its tasks' milestones
// and upserts a performance_cache entry with entity_type='PROJECT'.
// Called immediately after any milestone update so the project listing stays current.
func (d *PerformanceDao) RefreshForProject(projectID uint) error {
	taskDao := TaskDao{}
	milestoneDao := MilestoneDao{}

	tasks, _, err := taskDao.ListByProject(projectID, 0, 0)
	if err != nil {
		return err
	}

	var (
		sumExec, sumGoal, totalWeight float64
		totalMs, doneMs, blockedMs    int
		completedTasks                int
	)

	for _, t := range tasks {
		milestones, _ := milestoneDao.GetNonBlockedByTask(t.ID)
		var planned, achieved float64
		for _, m := range milestones {
			planned += m.PlannedValue
			achieved += m.AchievedValue
		}
		exec := util.ComputeExecutionScore(planned, achieved)

		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		goal := util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)

		w := t.Weight
		if w <= 0 {
			w = 100
		}
		sumExec += exec * w
		sumGoal += goal * w
		totalWeight += w

		tm, dm, bm := milestoneDao.CountByTask(t.ID)
		totalMs += int(tm)
		doneMs += int(dm)
		blockedMs += int(bm)
		if t.Status == "COMPLETED" {
			completedTasks++
		}
	}

	var execScore, goalScore float64
	if totalWeight > 0 {
		execScore = sumExec / totalWeight
		goalScore = sumGoal / totalWeight
	}
	totalScore := util.ComputePerformanceScore(execScore, goalScore)

	now := time.Now()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	cache := &model.PerformanceCache{
		EntityType:        "PROJECT",
		EntityID:          projectID,
		Period:            period,
		ExecutionScore:    execScore,
		GoalScore:         goalScore,
		TotalScore:        totalScore,
		TasksTotal:        len(tasks),
		TasksCompleted:    completedTasks,
		MilestonesTotal:   totalMs,
		MilestonesDone:    doneMs,
		MilestonesBlocked: blockedMs,
	}

	return d.Upsert(cache)
}

// ComputeScoreForScope aggregates execution+goal scores across all tasks that cover a given ASC or Regiao.
func (d *PerformanceDao) ComputeScoreForScope(scopeType string, scopeID uint) (execScore, goalScore, totalScore float64, trafficLight string) {
	taskDao := TaskDao{}
	milestoneDao := MilestoneDao{}

	tasks, err := taskDao.GetByScopeEntity(scopeType, scopeID)
	if err != nil || len(tasks) == 0 {
		return 0, 0, 0, "RED"
	}

	var sumExec, sumGoal, totalWeight float64
	for _, t := range tasks {
		milestones, _ := milestoneDao.GetNonBlockedByTask(t.ID)
		var planned, achieved float64
		for _, m := range milestones {
			planned += m.PlannedValue
			achieved += m.AchievedValue
		}
		exec := util.ComputeExecutionScore(planned, achieved)
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		goal := util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)
		w := t.Weight
		if w <= 0 {
			w = 100
		}
		sumExec += exec * w
		sumGoal += goal * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0, 0, 0, "RED"
	}
	execScore = sumExec / totalWeight
	goalScore = sumGoal / totalWeight
	totalScore = util.ComputePerformanceScore(execScore, goalScore)
	trafficLight = util.GetTrafficLight(totalScore)
	return
}

// ComputeScoreForScopeScoped is like ComputeScoreForScope but only considers
// tasks that are owned by the entities in the given UserScope.
// Use this when you want "how is this ASC performing relative to MY tasks".
// Falls back to ComputeScoreForScope for global users or regional directors.
func (d *PerformanceDao) ComputeScoreForScopeScoped(scopeType string, scopeID uint, scope *UserScope) (execScore, goalScore, totalScore float64, trafficLight string) {
	if scope.IsGlobal || scope.RegiaoID != 0 {
		return d.ComputeScoreForScope(scopeType, scopeID)
	}

	taskDao := TaskDao{}
	milestoneDao := MilestoneDao{}

	tasks, err := taskDao.GetByScopeEntityInScope(scopeType, scopeID, scope)
	if err != nil || len(tasks) == 0 {
		return 0, 0, 0, "RED"
	}

	var sumExec, sumGoal, totalWeight float64
	for _, t := range tasks {
		milestones, _ := milestoneDao.GetNonBlockedByTask(t.ID)
		var planned, achieved float64
		for _, m := range milestones {
			planned += m.PlannedValue
			achieved += m.AchievedValue
		}
		exec := util.ComputeExecutionScore(planned, achieved)
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		goal := util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)
		w := t.Weight
		if w <= 0 {
			w = 100
		}
		sumExec += exec * w
		sumGoal += goal * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0, 0, 0, "RED"
	}
	execScore = sumExec / totalWeight
	goalScore = sumGoal / totalWeight
	totalScore = util.ComputePerformanceScore(execScore, goalScore)
	trafficLight = util.GetTrafficLight(totalScore)
	return
}

// ComputeScoreForRegiao aggregates scores as the average of its child ASCs.
func (d *PerformanceDao) ComputeScoreForRegiao(regiaoID uint) (execScore, goalScore, totalScore float64, trafficLight string) {
	geoDao := GeoDao{}
	ascs, err := geoDao.ListASCsByRegiao(regiaoID)
	if err != nil || len(ascs) == 0 {
		// Fallback: compute directly from tasks with REGIAO scope
		return d.ComputeScoreForScope("REGIAO", regiaoID)
	}
	var sumExec, sumGoal float64
	for _, a := range ascs {
		e, g, _, _ := d.ComputeScoreForScope("ASC", a.ID)
		sumExec += e
		sumGoal += g
	}
	n := float64(len(ascs))
	execScore = sumExec / n
	goalScore = sumGoal / n
	totalScore = util.ComputePerformanceScore(execScore, goalScore)
	trafficLight = util.GetTrafficLight(totalScore)
	return
}

// ComputeScoreForOwner computes live scores for tasks owned by a given entity
// (owner_type = DIRECAO | DEPARTAMENTO, owner_id = id).
func (d *PerformanceDao) ComputeScoreForOwner(ownerType string, ownerID uint) (execScore, goalScore, totalScore float64, trafficLight string) {
	milestoneDao := MilestoneDao{}

	var tasks []model.Task
	Database.Preload("Milestones").
		Where("owner_type = ? AND owner_id = ? AND deleted_at IS NULL", ownerType, ownerID).
		Find(&tasks)

	if len(tasks) == 0 {
		return 0, 0, 0, "RED"
	}

	var sumExec, sumGoal, totalWeight float64
	for _, t := range tasks {
		milestones, _ := milestoneDao.GetNonBlockedByTask(t.ID)
		var planned, achieved float64
		for _, m := range milestones {
			planned += m.PlannedValue
			achieved += m.AchievedValue
		}
		exec := util.ComputeExecutionScore(planned, achieved)
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		goal := util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)
		w := t.Weight
		if w <= 0 {
			w = 100
		}
		sumExec += exec * w
		sumGoal += goal * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0, 0, 0, "RED"
	}
	execScore = sumExec / totalWeight
	goalScore = sumGoal / totalWeight
	totalScore = util.ComputePerformanceScore(execScore, goalScore)
	trafficLight = util.GetTrafficLight(totalScore)
	return
}

// LiveRankedItem is a computed performance entry for one entity.
type LiveRankedItem struct {
	EntityID     uint
	EntityName   string
	ExecScore    float64
	GoalScore    float64
	TotalScore   float64
	TrafficLight string
}

// LiveTopPerformers computes live rankings for any entity type.
func (d *PerformanceDao) LiveTopPerformers(entityType string, limit int) ([]LiveRankedItem, error) {
	geoDao := GeoDao{}
	var items []LiveRankedItem

	switch entityType {
	case "ASC":
		ascs, err := geoDao.GetAllASCs()
		if err != nil {
			return nil, err
		}
		for _, a := range ascs {
			e, g, t, l := d.ComputeScoreForScope("ASC", a.ID)
			items = append(items, LiveRankedItem{EntityID: a.ID, EntityName: a.Name, ExecScore: e, GoalScore: g, TotalScore: t, TrafficLight: l})
		}
	case "REGIAO":
		regioes, err := geoDao.GetAllRegioes()
		if err != nil {
			return nil, err
		}
		for _, r := range regioes {
			e, g, t, l := d.ComputeScoreForRegiao(r.ID)
			items = append(items, LiveRankedItem{EntityID: r.ID, EntityName: r.Name, ExecScore: e, GoalScore: g, TotalScore: t, TrafficLight: l})
		}
	case "DIRECAO":
		direcaoDao := DirecaoDao{}
		dirs, _, err := direcaoDao.List(0, 200)
		if err != nil {
			return nil, err
		}
		for _, dir := range dirs {
			e, g, t, l := d.ComputeScoreForOwner("DIRECAO", dir.ID)
			items = append(items, LiveRankedItem{EntityID: dir.ID, EntityName: dir.Name, ExecScore: e, GoalScore: g, TotalScore: t, TrafficLight: l})
		}
	case "DEPARTAMENTO":
		deptDao := DepartamentoDao{}
		depts, _, err := deptDao.List(0, 200)
		if err != nil {
			return nil, err
		}
		for _, dept := range depts {
			e, g, t, l := d.ComputeScoreForOwner("DEPARTAMENTO", dept.ID)
			items = append(items, LiveRankedItem{EntityID: dept.ID, EntityName: dept.Name, ExecScore: e, GoalScore: g, TotalScore: t, TrafficLight: l})
		}
	}

	// Sort descending
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].TotalScore > items[i].TotalScore {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// EmployeeScore holds computed performance for a single user.
type EmployeeScore struct {
	UserID       uint
	Name         string
	Role         string
	DeptName     string
	Category     string // DIR_DIRECAO | CHEFE_DEPT | DIR_ASC | DIR_REGIONAL | COLABORADOR
	ExecScore    float64
	GoalScore    float64
	TotalScore   float64
	TrafficLight string
	MsTotal      int
	MsDone       int
}

// EmployeeRanking computes performance for users in scope:
//   - role="CA"          → all non-admin users
//   - role="DIRECAO"     → users in depts under the given direcaoID
//   - role="DEPARTAMENTO"→ users in the given departamentoID
func (d *PerformanceDao) EmployeeRanking(role string, orgID uint) ([]EmployeeScore, error) {
	milestoneDao := MilestoneDao{}

	type userRow struct {
		ID       uint
		Name     string
		Role     string
		DeptID   uint
		DeptName string
	}

	var rows []userRow
	switch role {
	case "CA":
		Database.Raw(`
			SELECT u.id, u.name, u.role, COALESCE(du.departamento_id,0) AS dept_id, COALESCE(d.name,'') AS dept_name
			FROM users u
			LEFT JOIN departamento_users du ON du.user_id = u.id
			LEFT JOIN departamentos d ON d.id = du.departamento_id
			WHERE u.deleted_at IS NULL AND u.role NOT IN ('ADMIN','CA')
			ORDER BY u.name
		`).Scan(&rows)
	case "DIRECAO":
		Database.Raw(`
			SELECT u.id, u.name, u.role, COALESCE(du.departamento_id,0) AS dept_id, COALESCE(d.name,'') AS dept_name
			FROM users u
			JOIN departamento_users du ON du.user_id = u.id
			JOIN departamentos d ON d.id = du.departamento_id AND d.direcao_id = ? AND d.deleted_at IS NULL
			WHERE u.deleted_at IS NULL
			ORDER BY u.name
		`, orgID).Scan(&rows)
	case "DEPARTAMENTO":
		Database.Raw(`
			SELECT u.id, u.name, u.role, du.departamento_id AS dept_id, COALESCE(d.name,'') AS dept_name
			FROM users u
			JOIN departamento_users du ON du.user_id = u.id AND du.departamento_id = ?
			LEFT JOIN departamentos d ON d.id = du.departamento_id
			WHERE u.deleted_at IS NULL
			ORDER BY u.name
		`, orgID).Scan(&rows)
	default:
		return nil, nil
	}

	// Build userID → category map from org entity assignments
	type catRow struct{ UserID uint }
	categoryMap := map[uint]string{}

	var dirRows, deptRows, ascRows, regiaoRows []catRow
	Database.Raw(`SELECT DISTINCT responsible_id AS user_id FROM direcaos WHERE deleted_at IS NULL AND responsible_id IS NOT NULL`).Scan(&dirRows)
	Database.Raw(`SELECT DISTINCT responsible_id AS user_id FROM departamentos WHERE deleted_at IS NULL AND responsible_id IS NOT NULL`).Scan(&deptRows)
	Database.Raw(`SELECT DISTINCT director_id AS user_id FROM ascs WHERE deleted_at IS NULL AND director_id IS NOT NULL`).Scan(&ascRows)
	Database.Raw(`SELECT DISTINCT responsible_id AS user_id FROM regiaos WHERE deleted_at IS NULL AND responsible_id IS NOT NULL`).Scan(&regiaoRows)

	// Priority: dir > dept > asc > regional > colaborador
	for _, r := range regiaoRows { categoryMap[r.UserID] = "DIR_REGIONAL" }
	for _, r := range ascRows    { categoryMap[r.UserID] = "DIR_ASC" }
	for _, r := range deptRows   { categoryMap[r.UserID] = "CHEFE_DEPT" }
	for _, r := range dirRows    { categoryMap[r.UserID] = "DIR_DIRECAO" }

	var scores []EmployeeScore
	for _, r := range rows {
		// Milestones this user created or was the last updater of
		type msRow struct {
			PlannedValue  float64
			AchievedValue float64
			Status        string
			TaskID        uint
		}
		var ms []msRow
		Database.Raw(`
			SELECT m.planned_value, m.achieved_value, m.status, m.task_id
			FROM milestones m
			WHERE m.deleted_at IS NULL AND (m.created_by = ? OR m.updated_by = ?)
		`, r.ID, r.ID).Scan(&ms)

		cat := categoryMap[r.ID]
		if cat == "" {
			cat = "COLABORADOR"
		}

		if len(ms) == 0 {
			scores = append(scores, EmployeeScore{
				UserID: r.ID, Name: r.Name, Role: r.Role, DeptName: r.DeptName,
				Category: cat, TrafficLight: "RED",
			})
			continue
		}

		var totalPlanned, totalAchieved float64
		var msDone int
		taskIDs := map[uint]bool{}
		for _, m := range ms {
			totalPlanned += m.PlannedValue
			totalAchieved += m.AchievedValue
			if m.Status == "DONE" {
				msDone++
			}
			taskIDs[m.TaskID] = true
		}

		exec := util.ComputeExecutionScore(totalPlanned, totalAchieved)

		// Goal score: average of parent tasks' goal scores
		var sumGoal float64
		for taskID := range taskIDs {
			var t model.Task
			if err := Database.First(&t, taskID).Error; err != nil {
				continue
			}
			_ = milestoneDao // suppress unused warning
			startVal := float64(0)
			if t.StartValue != nil {
				startVal = *t.StartValue
			}
			sumGoal += util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)
		}
		goalScore := sumGoal / float64(len(taskIDs))
		totalScore := util.ComputePerformanceScore(exec, goalScore)

		scores = append(scores, EmployeeScore{
			UserID:       r.ID,
			Name:         r.Name,
			Role:         r.Role,
			DeptName:     r.DeptName,
			Category:     cat,
			ExecScore:    exec,
			GoalScore:    goalScore,
			TotalScore:   totalScore,
			TrafficLight: util.GetTrafficLight(totalScore),
			MsTotal:      len(ms),
			MsDone:       msDone,
		})
	}

	// Sort descending by TotalScore
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].TotalScore > scores[i].TotalScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	return scores, nil
}

// ScoreForUser computes the personal performance score for a single user based
// on the milestones they created or last-updated, regardless of department.
func (d *PerformanceDao) ScoreForUser(userID uint) EmployeeScore {
	milestoneDao := MilestoneDao{}

	// Basic user info
	var u model.User
	if err := Database.First(&u, userID).Error; err != nil {
		return EmployeeScore{UserID: userID, TrafficLight: "RED"}
	}

	type msRow struct {
		PlannedValue  float64
		AchievedValue float64
		Status        string
		TaskID        uint
	}
	var ms []msRow
	Database.Raw(`
		SELECT m.planned_value, m.achieved_value, m.status, m.task_id
		FROM milestones m
		WHERE m.deleted_at IS NULL AND (m.created_by = ? OR m.updated_by = ?)
	`, userID, userID).Scan(&ms)

	if len(ms) == 0 {
		return EmployeeScore{
			UserID: userID, Name: u.Name, Role: string(u.Role),
			Category: "DIR_DIRECAO", TrafficLight: "RED",
		}
	}

	var totalPlanned, totalAchieved float64
	var msDone int
	taskIDs := map[uint]bool{}
	for _, m := range ms {
		totalPlanned += m.PlannedValue
		totalAchieved += m.AchievedValue
		if m.Status == "DONE" {
			msDone++
		}
		taskIDs[m.TaskID] = true
	}

	exec := util.ComputeExecutionScore(totalPlanned, totalAchieved)

	var sumGoal float64
	for taskID := range taskIDs {
		var t model.Task
		if err := Database.First(&t, taskID).Error; err != nil {
			continue
		}
		_ = milestoneDao
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		sumGoal += util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)
	}
	goalScore := sumGoal / float64(len(taskIDs))
	totalScore := util.ComputePerformanceScore(exec, goalScore)

	return EmployeeScore{
		UserID:       userID,
		Name:         u.Name,
		Role:         string(u.Role),
		Category:     "DIR_DIRECAO",
		ExecScore:    exec,
		GoalScore:    goalScore,
		TotalScore:   totalScore,
		TrafficLight: util.GetTrafficLight(totalScore),
		MsTotal:      len(ms),
		MsDone:       msDone,
	}
}

// PerformanceCacheAlias is used by dashboard controller for map lookups
type PerformanceCacheAlias = model.PerformanceCache

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
