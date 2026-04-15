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

type ProjectController struct{}

func canAccessProject(scope *dao.UserScope, projectID uint) bool {
	if scope.IsGlobal {
		return true
	}

	var count int64
	scope.ApplyToProjects(
		dao.Database.Model(&model.Project{}).Where("id = ?", projectID),
	).Count(&count)

	return count > 0
}

// attachProjectPerf computes performance scores from each project's own
// start_value / current_value / target_value and injects them as a
// "performance" field so the frontend can render execution/goal bars.
func attachProjectPerf(projects []model.Project) []gin.H {
	out := make([]gin.H, len(projects))
	for i, p := range projects {
		m := projectToMap(p)

		var goalScore, execScore float64

		// Goal score from the project's own KPI fields
		if p.StartValue != nil && p.TargetValue != nil && p.CurrentValue != nil {
			goalScore = util.ComputeGoalScore(*p.StartValue, *p.TargetValue, *p.CurrentValue)
		}

		// Execution score from the project's tasks' milestones
		// Weighted average of per-task execution scores (matching taskExecScore logic)
		var sumExec, totalWeight float64
		for _, t := range p.Tasks {
			var nonBlocked []model.Milestone
			for _, ms := range t.Milestones {
				if ms.Status != "BLOCKED" {
					nonBlocked = append(nonBlocked, ms)
				}
			}
			if len(nonBlocked) == 0 {
				continue
			}

			var taskPlanned, taskAchieved float64
			switch t.AggregationType {
			case "AVG":
				for _, ms := range nonBlocked {
					taskPlanned += ms.PlannedValue
					taskAchieved += ms.AchievedValue
				}
				n := float64(len(nonBlocked))
				taskPlanned = taskPlanned / n
				taskAchieved = taskAchieved / n
			case "LAST":
				latest := nonBlocked[0]
				for _, ms := range nonBlocked[1:] {
					if ms.PlannedDate.After(latest.PlannedDate) {
						latest = ms
					} else if ms.PlannedDate.Equal(latest.PlannedDate) && ms.UpdatedAt.After(latest.UpdatedAt) {
						latest = ms
					}
				}
				taskPlanned = latest.PlannedValue
				taskAchieved = latest.AchievedValue
			default: // SUM_UP, SUM_DOWN, MANUAL
				for _, ms := range nonBlocked {
					taskPlanned += ms.PlannedValue
					taskAchieved += ms.AchievedValue
				}
			}

			// Detect reduction goal
			var taskExec float64
			taskStartVal := float64(0)
			if t.StartValue != nil {
				taskStartVal = *t.StartValue
			}
			if t.TargetValue < taskStartVal {
				taskExec = util.ComputeExecutionScoreReduction(taskPlanned, taskAchieved)
			} else {
				taskExec = util.ComputeExecutionScore(taskPlanned, taskAchieved)
			}

			w := t.Weight
			if w <= 0 {
				w = 100
			}
			sumExec += taskExec * w
			totalWeight += w
		}
		if totalWeight > 0 {
			execScore = sumExec / totalWeight
		}

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

// projectToMap converts a model.Project to a gin.H, preserving all existing JSON fields.
func projectToMap(p model.Project) gin.H {
	return gin.H{
		"id":             p.ID,
		"created_at":     p.CreatedAt,
		"updated_at":     p.UpdatedAt,
		"title":          p.Title,
		"description":    p.Description,
		"creator_type":   p.CreatorType,
		"creator_org_id": p.CreatorOrgID,
		"parent_id":      p.ParentID,
		"parent":         p.Parent,
		"weight":         p.Weight,
		"status":         p.Status,
		"created_by":     p.CreatedBy,
		"creator":        p.Creator,
		"start_date":     p.StartDate,
		"end_date":       p.EndDate,
		"direcoes":       p.Direcoes,
		"goal_label":     p.GoalLabel,
		"frequency":      p.Frequency,
		"start_value":    p.StartValue,
		"target_value":   p.TargetValue,
		"current_value":  p.CurrentValue,
	}
}

func (ProjectController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	filters := make(map[string]interface{})

	if ct := c.Query("creator_type"); ct != "" {
		filters["creator_type"] = ct
	}
	if pid := c.Query("parent_id"); pid != "" {
		id, _ := strconv.Atoi(pid)
		filters["parent_id"] = id
	}
	if st := c.Query("status"); st != "" {
		filters["status"] = st
	}

	// direcao_id filter: bypass normal scoped list and query via join table
	if didStr := c.Query("direcao_id"); didStr != "" {
		did, _ := strconv.Atoi(didStr)
		projectDao := dao.ProjectDao{}
		list, err := projectDao.ListByDirecao(uint(did))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		out := attachProjectPerf(list)
		c.JSON(http.StatusOK, util.NewPaginatedResponse(out, int64(len(out)), params))
		return
	}

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	projectDao := dao.ProjectDao{}
	list, total, err := projectDao.ListScoped(params.Page, params.Limit, filters, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	out := attachProjectPerf(list)
	c.JSON(http.StatusOK, util.NewPaginatedResponse(out, total, params))
}

type ProjectInput struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	CreatorType  string   `json:"creator_type" binding:"required,oneof=ADMIN CA PELOURO DIRECAO DEPARTAMENTO"`
	CreatorOrgID *uint    `json:"creator_org_id"`
	ParentID     *uint    `json:"parent_id"`
	Weight       float64  `json:"weight"`
	StartDate    *string  `json:"start_date"`
	EndDate      *string  `json:"end_date"`
	DirecaoIDs   []uint   `json:"direcao_ids"`
	Status       string   `json:"status"`        // optional: ACTIVE | COMPLETED | CANCELLED
	GoalLabel    string   `json:"goal_label"`    // e.g. "Perdas comerciais"
	Frequency    string   `json:"frequency"`     // DAILY,WEEKLY,MONTHLY,QUARTERLY,BIANNUAL,ANNUAL
	StartValue   *float64 `json:"start_value"`   // baseline
	TargetValue  *float64 `json:"target_value"`  // goal
	CurrentValue *float64 `json:"current_value"` // latest reported value
}

func (ProjectController) Create(c *gin.Context) {
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "department users cannot create projects"})
		return
	}

	var input ProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	project := model.Project{
		Title:        input.Title,
		Description:  input.Description,
		CreatorType:  input.CreatorType,
		CreatorOrgID: input.CreatorOrgID,
		ParentID:     input.ParentID,
		Weight:       input.Weight,
		Status:       "ACTIVE",
		CreatedBy:    util.ExtractUserID(c),
		GoalLabel:    input.GoalLabel,
		Frequency:    input.Frequency,
		StartValue:   input.StartValue,
		TargetValue:  input.TargetValue,
		CurrentValue: input.StartValue, // initialise current_value to the baseline
	}

	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		project.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		project.EndDate = t
	}

	if project.Weight == 0 {
		project.Weight = 100.0
	}

	projectDao := dao.ProjectDao{}
	if err := projectDao.Create(&project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Associate direcoes
	direcaoIDs := input.DirecaoIDs
	// Auto-populate from creator_org_id for DIRECAO-level projects
	if len(direcaoIDs) == 0 && input.CreatorType == "DIRECAO" && input.CreatorOrgID != nil {
		direcaoIDs = []uint{*input.CreatorOrgID}
	}
	if len(direcaoIDs) > 0 {
		if err := projectDao.SetDirecoes(project.ID, direcaoIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
	}

	// Reload with associations
	project, _ = projectDao.GetByID(project.ID)

	auditDao := dao.AuditDao{}
	auditDao.Write("project", project.ID, util.ExtractUserID(c), "CREATE", nil, project, c.ClientIP())

	c.JSON(http.StatusCreated, project)
}

func (ProjectController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !canAccessProject(&scope, uint(id)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, project)
}

func (ProjectController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "department users cannot update projects"})
		return
	}

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !canAccessProject(&scope, uint(id)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := project

	var input ProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	project.Title = input.Title
	project.Description = input.Description
	project.CreatorType = input.CreatorType
	project.CreatorOrgID = input.CreatorOrgID
	project.ParentID = input.ParentID
	if input.Weight > 0 {
		project.Weight = input.Weight
	}
	if input.Status != "" {
		project.Status = input.Status
	}
	if input.GoalLabel != "" {
		project.GoalLabel = input.GoalLabel
	}
	if input.Frequency != "" {
		project.Frequency = input.Frequency
	}
	if input.StartValue != nil {
		project.StartValue = input.StartValue
	}
	if input.TargetValue != nil {
		project.TargetValue = input.TargetValue
	}
	if input.CurrentValue != nil {
		project.CurrentValue = input.CurrentValue
	}

	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		project.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		project.EndDate = t
	}

	if err := projectDao.Update(&project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Replace direcoes association (empty slice clears all)
	if err := projectDao.SetDirecoes(project.ID, input.DirecaoIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Reload with associations
	project, _ = projectDao.GetByID(project.ID)

	auditDao := dao.AuditDao{}
	auditDao.Write("project", project.ID, util.ExtractUserID(c), "UPDATE", oldData, project, c.ClientIP())

	c.JSON(http.StatusOK, project)
}

// UpdateProgress handles PATCH /projects/:id/progress
// Allows the responsible director to update the project's current_value.
func (ProjectController) UpdateProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "department users cannot update project progress"})
		return
	}

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !canAccessProject(&scope, uint(id)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	var input struct {
		CurrentValue    float64 `json:"current_value" binding:"required"`
		PeriodReference string  `json:"period_reference"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	cv := input.CurrentValue
	project.CurrentValue = &cv

	if err := projectDao.Update(&project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Auto-create a history entry so the dashboard chart tracks real value evolution
	periodRef := input.PeriodReference
	if periodRef == "" {
		now := time.Now()
		periodRef = fmt.Sprintf("%d-%02d", now.Year(), now.Month())
	}
	historyDao := dao.ProjectHistoryDao{}
	_ = historyDao.Create(&model.ProjectHistory{
		ProjectID:       project.ID,
		Value:           cv,
		PeriodReference: periodRef,
		CreatedBy:       util.ExtractUserID(c),
	})

	project, _ = projectDao.GetByID(project.ID)

	auditDao := dao.AuditDao{}
	auditDao.Write("project", project.ID, util.ExtractUserID(c), "UPDATE_PROGRESS", nil, project, c.ClientIP())

	c.JSON(http.StatusOK, project)
}

func (ProjectController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if util.ExtractRole(c) == "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "department users cannot delete projects"})
		return
	}

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !canAccessProject(&scope, uint(id)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	projectDao := dao.ProjectDao{}

	project, err := projectDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := projectDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("project", uint(id), util.ExtractUserID(c), "DELETE", project, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (ProjectController) Tree(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !canAccessProject(&scope, uint(id)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetTree(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, project)
}
