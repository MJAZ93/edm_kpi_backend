package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type TaskController struct{}

func (TaskController) ListByProject(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	params := util.ParsePagination(c)
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	taskDao := dao.TaskDao{}
	list, total, err := taskDao.ListByProjectScoped(uint(projectID), params.Page, params.Limit, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	enriched := attachTaskPerf(list)
	c.JSON(http.StatusOK, util.NewPaginatedResponse(enriched, total, params))
}

// attachTaskPerf computes live execution/goal/total scores for each task
// from its milestones and injects a "performance" field into the JSON output.
func attachTaskPerf(tasks []model.Task) []gin.H {
	out := make([]gin.H, len(tasks))
	for i, t := range tasks {
		m := taskToMap(t)

		// Filter non-blocked milestones
		var nonBlocked []model.Milestone
		for _, ms := range t.Milestones {
			if ms.Status != "BLOCKED" {
				nonBlocked = append(nonBlocked, ms)
			}
		}

		var execScore, goalScore float64

		if len(nonBlocked) > 0 {
			var planned, achieved float64
			switch t.AggregationType {
			case "AVG":
				for _, ms := range nonBlocked {
					planned += ms.PlannedValue
					achieved += ms.AchievedValue
				}
				n := float64(len(nonBlocked))
				planned /= n
				achieved /= n
			case "LAST":
				latest := nonBlocked[0]
				for _, ms := range nonBlocked[1:] {
					if ms.PlannedDate.After(latest.PlannedDate) {
						latest = ms
					} else if ms.PlannedDate.Equal(latest.PlannedDate) && ms.UpdatedAt.After(latest.UpdatedAt) {
						latest = ms
					}
				}
				planned = latest.PlannedValue
				achieved = latest.AchievedValue
			default: // SUM_UP, SUM_DOWN, MANUAL
				for _, ms := range nonBlocked {
					planned += ms.PlannedValue
					achieved += ms.AchievedValue
				}
			}

			startVal := float64(0)
			if t.StartValue != nil {
				startVal = *t.StartValue
			}
			if t.TargetValue < startVal {
				execScore = util.ComputeExecutionScoreReduction(planned, achieved)
			} else {
				execScore = util.ComputeExecutionScore(planned, achieved)
			}
		}

		// Goal score from the task's own KPI fields
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		goalScore = util.ComputeGoalScore(startVal, t.TargetValue, t.CurrentValue)

		totalScore := util.ComputePerformanceScore(execScore, goalScore)

		m["performance"] = gin.H{
			"execution_score": execScore,
			"goal_score":      goalScore,
			"total_score":     totalScore,
			"traffic_light":   util.GetTrafficLight(totalScore),
		}
		out[i] = m
	}
	return out
}

// taskToMap converts a model.Task to gin.H preserving JSON fields.
func taskToMap(t model.Task) gin.H {
	m := gin.H{
		"id":               t.ID,
		"created_at":       t.CreatedAt,
		"updated_at":       t.UpdatedAt,
		"project_id":       t.ProjectID,
		"parent_task_id":   t.ParentTaskID,
		"title":            t.Title,
		"description":      t.Description,
		"owner_type":       t.OwnerType,
		"owner_id":         t.OwnerID,
		"frequency":        t.Frequency,
		"goal_label":       t.GoalLabel,
		"start_value":      t.StartValue,
		"target_value":     t.TargetValue,
		"current_value":    t.CurrentValue,
		"aggregation_type": t.AggregationType,
		"weight":           t.Weight,
		"start_date":       t.StartDate,
		"end_date":         t.EndDate,
		"next_update_due":  t.NextUpdateDue,
		"status":           t.Status,
		"created_by":       t.CreatedBy,
		"assigned_to":      t.AssignedTo,
	}
	if t.Creator != nil {
		m["creator"] = t.Creator
	}
	if t.Assignee != nil {
		m["assignee"] = t.Assignee
	}
	if len(t.Scopes) > 0 {
		m["scopes"] = t.Scopes
	}
	return m
}

type TaskInput struct {
	Title           string       `json:"title" binding:"required"`
	Description     string       `json:"description"`
	OwnerType       string       `json:"owner_type" binding:"required,oneof=DIRECAO DEPARTAMENTO"`
	OwnerID         uint         `json:"owner_id" binding:"required"`
	AssignedTo      *uint        `json:"assigned_to"`
	Frequency       string       `json:"frequency" binding:"required,oneof=DAILY WEEKLY MONTHLY QUARTERLY BIANNUAL ANNUAL"`
	GoalLabel       string       `json:"goal_label"`
	StartValue      *float64     `json:"start_value"`
	TargetValue     float64      `json:"target_value" binding:"required"`
	AggregationType string       `json:"aggregation_type"` // SUM_UP, SUM_DOWN, AVG, LAST, MANUAL
	CurrentValue    *float64     `json:"current_value"`    // only used when aggregation_type = MANUAL
	Weight          float64      `json:"weight"`
	StartDate       *string      `json:"start_date"`
	EndDate         *string      `json:"end_date"`
	ParentTaskID    *uint        `json:"parent_task_id"`
	Scopes          []ScopeInput `json:"scopes"`
}

type ScopeInput struct {
	ScopeType string `json:"scope_type" binding:"required,oneof=NACIONAL REGIONAL ASC"`
	ScopeID   *uint  `json:"scope_id"`
}

func validateTaskAssignee(ownerType string, ownerID uint, assignedTo *uint) error {
	if assignedTo == nil {
		return nil
	}
	if ownerType != "DEPARTAMENTO" {
		return fmt.Errorf("assigned_to requires owner_type DEPARTAMENTO")
	}

	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByID(ownerID)
	if err != nil {
		return fmt.Errorf("department not found")
	}

	if dept.ResponsibleID != nil && *dept.ResponsibleID == *assignedTo {
		return nil
	}
	for _, u := range dept.Users {
		if u.ID == *assignedTo {
			return nil
		}
	}

	return fmt.Errorf("assigned user must belong to selected department")
}

func (TaskController) Create(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))

	var input TaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	if err := validateTaskAssignee(input.OwnerType, input.OwnerID, input.AssignedTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	// DEPARTAMENTO users: only dept heads can create tasks.
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		userID := util.ExtractUserID(c)
		if !dao.IsDeptHead(userID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "técnicos não podem criar acções"})
			return
		}
		scope := dao.ResolveScope(userID, "DEPARTAMENTO")
		ownDept := false
		for _, id := range scope.DepartamentoIDs {
			if id == input.OwnerID && input.OwnerType == "DEPARTAMENTO" {
				ownDept = true
				break
			}
		}
		if !ownDept {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "só pode criar acções para o seu departamento"})
			return
		}
	}

	aggType := input.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	task := model.Task{
		ProjectID:       uint(projectID),
		Title:           input.Title,
		Description:     input.Description,
		OwnerType:       input.OwnerType,
		OwnerID:         input.OwnerID,
		AssignedTo:      input.AssignedTo,
		Frequency:       input.Frequency,
		GoalLabel:       input.GoalLabel,
		StartValue:      input.StartValue,
		TargetValue:     input.TargetValue,
		AggregationType: aggType,
		Weight:          input.Weight,
		ParentTaskID:    input.ParentTaskID,
		Status:          "ACTIVE",
		CreatedBy:       util.ExtractUserID(c),
	}

	if task.Weight == 0 {
		task.Weight = 100.0
	}

	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		task.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		task.EndDate = t
	}

	taskDao := dao.TaskDao{}
	if err := taskDao.Create(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Create scopes
	if len(input.Scopes) > 0 {
		var scopes []model.TaskScope
		for _, s := range input.Scopes {
			scopes = append(scopes, model.TaskScope{
				TaskID:    task.ID,
				ScopeType: s.ScopeType,
				ScopeID:   s.ScopeID,
			})
		}
		taskDao.CreateScopes(scopes)
	}

	// Notify ASC directors for scoped tasks
	go notifyASCDirectorsForTask(task, input.Scopes, util.ExtractUserID(c))

	auditDao := dao.AuditDao{}
	auditDao.Write("task", task.ID, util.ExtractUserID(c), "CREATE", nil, task, c.ClientIP())

	// Reload with relations
	result, _ := taskDao.GetByID(task.ID)
	c.JSON(http.StatusCreated, result)
}

func (TaskController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// TaskUpdateInput is a partial-update struct — all fields are optional.
// Missing fields fall back to the existing task values.
type TaskUpdateInput struct {
	Title           *string      `json:"title"`
	Description     *string      `json:"description"`
	OwnerType       *string      `json:"owner_type"`
	OwnerID         *uint        `json:"owner_id"`
	AssignedTo      *uint        `json:"assigned_to"`
	Frequency       *string      `json:"frequency"`
	GoalLabel       *string      `json:"goal_label"`
	StartValue      *float64     `json:"start_value"`
	TargetValue     *float64     `json:"target_value"`
	AggregationType *string      `json:"aggregation_type"`
	CurrentValue    *float64     `json:"current_value"`
	Weight          *float64     `json:"weight"`
	StartDate       *string      `json:"start_date"`
	EndDate         *string      `json:"end_date"`
	ParentTaskID    *uint        `json:"parent_task_id"`
	Scopes          []ScopeInput `json:"scopes"`
}

func (TaskController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// Technicians (non-head DEPARTAMENTO) cannot update tasks
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		if !dao.IsDeptHead(util.ExtractUserID(c)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "técnicos não podem actualizar acções"})
			return
		}
	}

	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := task

	var input TaskUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	// Apply only the fields that were provided
	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.OwnerType != nil {
		task.OwnerType = *input.OwnerType
	}
	if input.OwnerID != nil {
		task.OwnerID = *input.OwnerID
	}
	if input.AssignedTo != nil {
		task.AssignedTo = input.AssignedTo
	}
	if input.Frequency != nil {
		task.Frequency = *input.Frequency
	}
	if input.GoalLabel != nil {
		task.GoalLabel = *input.GoalLabel
	}
	if input.StartValue != nil {
		task.StartValue = input.StartValue
	}
	if input.TargetValue != nil {
		task.TargetValue = *input.TargetValue
	}
	if input.AggregationType != nil && *input.AggregationType != "" {
		task.AggregationType = *input.AggregationType
	}
	if input.Weight != nil && *input.Weight > 0 {
		task.Weight = *input.Weight
	}
	if input.ParentTaskID != nil {
		task.ParentTaskID = input.ParentTaskID
	}
	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		task.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		task.EndDate = t
	}

	if err := validateTaskAssignee(task.OwnerType, task.OwnerID, task.AssignedTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	if err := taskDao.Update(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// If the task's date range changed, extend every milestone's monthly
	// rows to cover the new range (idempotent).
	if task.StartDate != nil && task.EndDate != nil {
		startChanged := oldData.StartDate == nil || !task.StartDate.Equal(*oldData.StartDate)
		endChanged := oldData.EndDate == nil || !task.EndDate.Equal(*oldData.EndDate)
		if startChanged || endChanged {
			mmt := dao.MilestoneMonthlyTargetDao{}
			go mmt.EnsureMonthsForTask(task.ID, *task.StartDate, *task.EndDate, util.ExtractUserID(c))
		}
	}

	// Handle current_value: MANUAL override or auto-recalc
	if task.AggregationType == "MANUAL" && input.CurrentValue != nil {
		task.CurrentValue = *input.CurrentValue
		taskDao.Update(&task)
		// Record progress history for MANUAL tasks
		histDao := dao.TaskProgressHistoryDao{}
		go histDao.UpsertProgress(task.ID, task.CurrentValue, util.ExtractUserID(c))
		perfDao := dao.PerformanceDao{}
		go perfDao.RefreshForTask(task.ID)
		go perfDao.RefreshForProject(task.ProjectID)
	} else if task.AggregationType != "MANUAL" {
		startChanged := (task.StartValue == nil) != (oldData.StartValue == nil) ||
			(task.StartValue != nil && oldData.StartValue != nil && *task.StartValue != *oldData.StartValue)
		if task.AggregationType != oldData.AggregationType || startChanged {
			taskDao.RecalcCurrentValue(task.ID)
			perfDao := dao.PerformanceDao{}
			go perfDao.RefreshForTask(task.ID)
			go perfDao.RefreshForProject(task.ProjectID)
		}
	}

	// Update scopes
	if len(input.Scopes) > 0 {
		taskDao.DeleteScopes(task.ID)
		var scopes []model.TaskScope
		for _, s := range input.Scopes {
			scopes = append(scopes, model.TaskScope{
				TaskID:    task.ID,
				ScopeType: s.ScopeType,
				ScopeID:   s.ScopeID,
			})
		}
		taskDao.CreateScopes(scopes)
	}

	// Notify up the chain
	go notifyTaskUpdateChain(task, util.ExtractUserID(c))

	auditDao := dao.AuditDao{}
	auditDao.Write("task", task.ID, util.ExtractUserID(c), "UPDATE", oldData, task, c.ClientIP())

	result, _ := taskDao.GetByID(task.ID)
	c.JSON(http.StatusOK, result)
}

// ListProgress handles GET /tasks/:id/progress
// Returns all monthly progress history entries for a task (period + value).
func (TaskController) ListProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	histDao := dao.TaskProgressHistoryDao{}
	entries, err := histDao.ListByTask(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// PatchProgress handles PATCH /tasks/:id/progress
// Allows updating a MANUAL task's current_value with a specific period reference.
func (TaskController) PatchProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if task.AggregationType != "MANUAL" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "only MANUAL tasks support manual progress updates"})
		return
	}

	var input struct {
		CurrentValue float64 `json:"current_value"` // 0 is a valid progress value
		Period       string  `json:"period"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	period := input.Period
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	task.CurrentValue = input.CurrentValue
	if err := taskDao.Update(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	histDao := dao.TaskProgressHistoryDao{}
	_ = histDao.UpsertProgressForPeriod(task.ID, period, input.CurrentValue, util.ExtractUserID(c))

	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(task.ID)
	go perfDao.RefreshForProject(task.ProjectID)

	auditDao := dao.AuditDao{}
	auditDao.Write("task", task.ID, util.ExtractUserID(c), "UPDATE_PROGRESS", nil,
		gin.H{"current_value": input.CurrentValue, "period": period}, c.ClientIP())

	result, _ := taskDao.GetByID(task.ID)
	c.JSON(http.StatusOK, result)
}

func (TaskController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	taskDao := dao.TaskDao{}

	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := taskDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("task", uint(id), util.ExtractUserID(c), "DELETE", task, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func notifyASCDirectorsForTask(task model.Task, scopes []ScopeInput, actorID uint) {
	geoDao := dao.GeoDao{}
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	actor, _ := userDao.GetByID(actorID)

	for _, s := range scopes {
		if s.ScopeType == "ASC" && s.ScopeID != nil {
			asc, err := geoDao.GetASCByID(*s.ScopeID)
			if err != nil || asc.DirectorID == nil {
				continue
			}
			director, err := userDao.GetByID(*asc.DirectorID)
			if err != nil {
				continue
			}
			notifDao.CreateAndEmail(director.ID, "Nova tarefa na sua ASC",
				"A tarefa '"+task.Title+"' foi atribuída à "+asc.Name, "TASK_UPDATE", "task", &task.ID)
			go util.EmailTaskUpdated(director.Email, director.Name, task.Title, actor.Name)
		}
	}
}

func notifyTaskUpdateChain(task model.Task, actorID uint) {
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}
	direcaoDao := dao.DirecaoDao{}
	pelouroDao := dao.PelouroDao{}

	actor, _ := userDao.GetByID(actorID)

	// Notify up: DEPARTAMENTO → DIRECAO → PELOURO → CA
	if task.OwnerType == "DEPARTAMENTO" {
		deptDao := dao.DepartamentoDao{}
		dept, err := deptDao.GetByID(task.OwnerID)
		if err == nil {
			// Notify Direção responsible
			direcao, err := direcaoDao.GetByID(dept.DirecaoID)
			if err == nil && direcao.ResponsibleID != nil {
				notifDao.CreateAndEmail(*direcao.ResponsibleID, "Tarefa actualizada",
					"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
				resp, _ := userDao.GetByID(*direcao.ResponsibleID)
				go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)

				// Notify Pelouro
				pelouro, err := pelouroDao.GetByID(direcao.PelouroID)
				if err == nil && pelouro.ResponsibleID != nil {
					notifDao.CreateAndEmail(*pelouro.ResponsibleID, "Tarefa actualizada",
						"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
					resp, _ := userDao.GetByID(*pelouro.ResponsibleID)
					go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)
				}
			}
		}
	} else if task.OwnerType == "DIRECAO" {
		direcao, err := direcaoDao.GetByID(task.OwnerID)
		if err == nil {
			pelouro, err := pelouroDao.GetByID(direcao.PelouroID)
			if err == nil && pelouro.ResponsibleID != nil {
				notifDao.CreateAndEmail(*pelouro.ResponsibleID, "Tarefa actualizada",
					"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
				resp, _ := userDao.GetByID(*pelouro.ResponsibleID)
				go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)
			}
		}
	}

	// Always notify CA and ADMIN users
	caUsers, _ := userDao.GetByRole("CA")
	adminUsers, _ := userDao.GetByRole("ADMIN")
	allSuperUsers := append(caUsers, adminUsers...)
	for _, su := range allSuperUsers {
		if su.ID != actorID {
			notifDao.CreateAndEmail(su.ID, "Tarefa actualizada",
				"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
			go util.EmailTaskUpdated(su.Email, su.Name, task.Title, actor.Name)
		}
	}
}
