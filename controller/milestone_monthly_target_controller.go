package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type MilestoneMonthlyTargetController struct{}

// canEditMilestoneMonthly returns true when the caller is allowed to edit
// monthly metas for a milestone. Allowed:
//   - ADMIN / CA (global roles)
//   - assigned user on the milestone or on the parent task
//   - Department head whose department owns the task
//   - Direcção / Pelouro head up the chain that covers the task's owner
func canEditMilestoneMonthly(callerID uint, callerRole string, msAssignedTo *uint, task model.Task) bool {
	// 1. Global admins / CA can do anything.
	if callerRole == "ADMIN" || callerRole == "CA" {
		return true
	}
	// 2. The assigned user on the milestone or the task.
	if msAssignedTo != nil && *msAssignedTo == callerID {
		return true
	}
	if task.AssignedTo != nil && *task.AssignedTo == callerID {
		return true
	}
	// 3. Department head whose department owns the task.
	if callerRole == "DEPARTAMENTO" && dao.IsDeptHead(callerID) && task.OwnerType == "DEPARTAMENTO" {
		scope := dao.ResolveScope(callerID, "DEPARTAMENTO")
		for _, id := range scope.DepartamentoIDs {
			if task.OwnerID == id {
				return true
			}
		}
	}
	// 4. Direcção head whose direcção owns (or contains the dept that owns) the task.
	if callerRole == "DIRECAO" {
		scope := dao.ResolveScope(callerID, "DIRECAO")
		if task.OwnerType == "DIRECAO" {
			for _, id := range scope.DirecaoIDs {
				if task.OwnerID == id {
					return true
				}
			}
		}
		if task.OwnerType == "DEPARTAMENTO" {
			for _, id := range scope.DepartamentoIDs {
				if task.OwnerID == id {
					return true
				}
			}
		}
	}
	// 5. Pelouro head — same idea extended up the chain.
	if callerRole == "PELOURO" {
		scope := dao.ResolveScope(callerID, "PELOURO")
		if task.OwnerType == "DIRECAO" {
			for _, id := range scope.DirecaoIDs {
				if task.OwnerID == id {
					return true
				}
			}
		}
		if task.OwnerType == "DEPARTAMENTO" {
			for _, id := range scope.DepartamentoIDs {
				if task.OwnerID == id {
					return true
				}
			}
		}
	}
	return false
}

// ListByMilestone returns all monthly target rows for a milestone.
// GET /private/milestones/:id/monthly-targets
func (MilestoneMonthlyTargetController) ListByMilestone(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	mmt := dao.MilestoneMonthlyTargetDao{}
	list, err := mmt.ListByMilestone(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": list, "total": len(list)})
}

type monthlyTargetInput struct {
	Period       string   `json:"period" binding:"required"`
	PlannedValue *float64 `json:"planned_value"`
	Notes        *string  `json:"notes"`
	// AchievedValue is intentionally NOT accepted from the client — it is
	// derived server-side from MilestoneProgress rows via
	// SyncAchievedFromProgress. Accepting it would let users bypass the
	// progress-update audit trail.
}

// UpsertRow creates or updates a single monthly target row.
// PUT /private/milestones/:id/monthly-targets
func (MilestoneMonthlyTargetController) UpsertRow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID := util.ExtractUserID(c)

	var input monthlyTargetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	// Verify milestone exists + permission
	milestoneDao := dao.MilestoneDao{}
	ms, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// Permission check: admins, global CA, assigned users, and department/
	// direcção/pelouro heads up the chain can edit.
	callerRole := util.ExtractRole(c)
	taskDao := dao.TaskDao{}
	task, _ := taskDao.GetByID(ms.TaskID)
	if !canEditMilestoneMonthly(userID, callerRole, ms.AssignedTo, task) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Apenas o responsável, chefe do departamento ou administrador pode editar as metas mensais."})
		return
	}

	mmt := dao.MilestoneMonthlyTargetDao{}

	// Validate: the sum of monthly planned values cannot exceed the
	// milestone's global planned_value (ms.PlannedValue stays immutable from
	// monthly edits — it's the committed target at creation).
	// Skip for AVG/MANUAL tasks: each month is an independent rate, not a slice
	// of the global total, so summing them is meaningless.
	isAveragingTask := task.AggregationType == "AVG" || task.AggregationType == "MANUAL"
	if !isAveragingTask && input.PlannedValue != nil && ms.PlannedValue > 0 {
		existingRows, _ := mmt.ListByMilestone(uint(id))
		var projectedSum float64
		for _, r := range existingRows {
			if r.Period == input.Period {
				projectedSum += *input.PlannedValue
			} else {
				projectedSum += r.PlannedValue
			}
		}
		// If the period wasn't in existing rows, add the new one
		found := false
		for _, r := range existingRows {
			if r.Period == input.Period {
				found = true
				break
			}
		}
		if !found {
			projectedSum += *input.PlannedValue
		}
		if projectedSum > ms.PlannedValue+0.001 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "monthly_sum_exceeds_global",
				"message": "A soma das metas mensais não pode ser maior que a meta total do indicador.",
			})
			return
		}
	}

	// Always pass nil for achieved — it stays as-is or starts at 0; the real
	// value lands via SyncAchievedFromProgress when progress is recorded.
	row, err := mmt.UpsertRow(uint(id), input.Period, input.PlannedValue, nil, input.Notes, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// NOTE: ms.PlannedValue (the global meta) is intentionally NOT overwritten
	// from the monthly rollup — the global target is the committed value the
	// user picked at creation time; monthly rows are a breakdown <= that sum.
	ms.UpdatedBy = &userID
	_ = milestoneDao.Update(&ms)

	// Cascade to task current_value
	if task.AggregationType != "MANUAL" {
		taskDao.RecalcCurrentValue(ms.TaskID)
		// Record task progress history for this period
		histDao := dao.TaskProgressHistoryDao{}
		// Parse period "YYYY-MM" to a time.Time for RecalcForPeriod
		if t, err := parseMonthlyPeriod(input.Period); err == nil {
			go histDao.RecalcForPeriod(ms.TaskID, t, userID)
		}
	}

	// Refresh performance cache for task + project
	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(ms.TaskID)
	if task.ProjectID != 0 {
		go perfDao.RefreshForProject(task.ProjectID)
	}

	c.JSON(http.StatusOK, row)
}

// BulkUpsert replaces / merges a batch of monthly rows for a milestone.
// PUT /private/milestones/:id/monthly-targets/bulk
func (MilestoneMonthlyTargetController) BulkUpsert(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID := util.ExtractUserID(c)

	var input struct {
		Rows []monthlyTargetInput `json:"rows" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	// Permission check
	milestoneDao := dao.MilestoneDao{}
	ms, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	callerRole := util.ExtractRole(c)
	taskDao := dao.TaskDao{}
	task, _ := taskDao.GetByID(ms.TaskID)
	if !canEditMilestoneMonthly(userID, callerRole, ms.AssignedTo, task) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Apenas o responsável, chefe do departamento ou administrador pode editar as metas mensais."})
		return
	}

	mmt := dao.MilestoneMonthlyTargetDao{}

	// Validate: the resulting sum of monthly planned values cannot exceed the
	// milestone's global planned_value. We simulate the upsert by merging the
	// input into the existing rows and summing.
	// Skip for AVG/MANUAL tasks: each month is an independent rate, not a slice
	// of the global total, so summing them is meaningless.
	isAveragingTask := task.AggregationType == "AVG" || task.AggregationType == "MANUAL"
	if !isAveragingTask && ms.PlannedValue > 0 {
		existingRows, _ := mmt.ListByMilestone(uint(id))
		planByPeriod := map[string]float64{}
		for _, r := range existingRows {
			planByPeriod[r.Period] = r.PlannedValue
		}
		for _, r := range input.Rows {
			if r.PlannedValue != nil {
				planByPeriod[r.Period] = *r.PlannedValue
			}
		}
		var projectedSum float64
		for _, v := range planByPeriod {
			projectedSum += v
		}
		if projectedSum > ms.PlannedValue+0.001 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "monthly_sum_exceeds_global",
				"message": "A soma das metas mensais não pode ser maior que a meta total do indicador.",
			})
			return
		}
	}

	for _, r := range input.Rows {
		// achieved_value is never accepted from the client — pass nil.
		_, _ = mmt.UpsertRow(uint(id), r.Period, r.PlannedValue, nil, r.Notes, userID)
	}

	// NOTE: ms.PlannedValue (global target) is intentionally NOT rolled up
	// from monthly rows — it's the committed value from milestone creation.
	// Monthly rows must sum to AT MOST that value (validated above).
	ms.UpdatedBy = &userID
	_ = milestoneDao.Update(&ms)

	// Cascade
	if task.AggregationType != "MANUAL" {
		taskDao.RecalcCurrentValue(ms.TaskID)
		histDao := dao.TaskProgressHistoryDao{}
		for _, r := range input.Rows {
			if t, err := parseMonthlyPeriod(r.Period); err == nil {
				go histDao.RecalcForPeriod(ms.TaskID, t, userID)
			}
		}
	}

	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(ms.TaskID)
	if task.ProjectID != 0 {
		go perfDao.RefreshForProject(task.ProjectID)
	}

	// Return the fresh list
	list, _ := mmt.ListByMilestone(uint(id))
	c.JSON(http.StatusOK, gin.H{"rows": list, "total": len(list)})
}

// MonthlyChartForTask returns aggregated monthly planned/achieved for a task.
// GET /private/tasks/:id/monthly-chart
func (MilestoneMonthlyTargetController) MonthlyChartForTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	mmt := dao.MilestoneMonthlyTargetDao{}
	rows, err := mmt.MonthlyForTask(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows})
}

// MonthlyChartForProject returns aggregated monthly planned/achieved for a project.
// GET /private/projects/:id/monthly-chart
func (MilestoneMonthlyTargetController) MonthlyChartForProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	mmt := dao.MilestoneMonthlyTargetDao{}
	rows, err := mmt.MonthlyForProject(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows})
}
