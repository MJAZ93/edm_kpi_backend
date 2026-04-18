package controller

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type ProjectHistoryController struct{}

type ProjectHistoryInput struct {
	Value           float64 `json:"value" binding:"required"`
	PeriodReference string  `json:"period_reference"`
	Notes           string  `json:"notes"`
}

func (ProjectHistoryController) Create(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))

	var input ProjectHistoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	// Verify project exists
	projectDao := dao.ProjectDao{}
	if _, err := projectDao.GetByID(uint(projectID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "project not found"})
		return
	}

	entry := model.ProjectHistory{
		ProjectID:       uint(projectID),
		Value:           input.Value,
		PeriodReference: input.PeriodReference,
		Notes:           input.Notes,
		CreatedBy:       util.ExtractUserID(c),
	}

	historyDao := dao.ProjectHistoryDao{}
	if err := historyDao.Create(&entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (ProjectHistoryController) List(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))

	historyDao := dao.ProjectHistoryDao{}
	list, err := historyDao.ListByProject(uint(projectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": list, "total": len(list)})
}

func (ProjectHistoryController) Update(c *gin.Context) {
	entryID, _ := strconv.Atoi(c.Param("entry_id"))

	historyDao := dao.ProjectHistoryDao{}
	entry, err := historyDao.GetByID(uint(entryID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	var input struct {
		Value           *float64 `json:"value"`
		PeriodReference *string  `json:"period_reference"`
		Notes           *string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	if input.Value != nil {
		entry.Value = *input.Value
	}
	if input.PeriodReference != nil {
		entry.PeriodReference = *input.PeriodReference
	}
	if input.Notes != nil {
		entry.Notes = *input.Notes
	}

	if err := historyDao.Update(&entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (ProjectHistoryController) Delete(c *gin.Context) {
	entryID, _ := strconv.Atoi(c.Param("entry_id"))

	historyDao := dao.ProjectHistoryDao{}
	if _, err := historyDao.GetByID(uint(entryID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := historyDao.Delete(uint(entryID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// milestoneTaskProgress computes the progress percentage (0–100) of a milestone
// toward its task's goal, using the same universal formula as ComputeGoalScore.
// Works for both growth goals (target > start) and reduction goals (target < start).
// formula: (achieved - start) / (target - start) * 100
func milestoneTaskProgress(achievedValue, startValue, targetValue float64) float64 {
	diff := targetValue - startValue
	if diff == 0 {
		return 100
	}
	progress := (achievedValue - startValue) / diff * 100
	return math.Max(0, math.Min(100, progress))
}

// ExecutionHistory returns per-period execution percentages for a project.
// Uses task_progress_histories for recorded months and task.current_value for
// any month that has no history yet (including the current month).
func (ProjectHistoryController) ExecutionHistory(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))

	// Get project for date range
	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// Fetch all task progress history records for this project
	type HistRow struct {
		TaskID      uint
		Period      string
		Value       float64
		StartValue  *float64
		TargetValue float64
	}
	var histRows []HistRow
	dao.Database.Raw(`
		SELECT h.task_id, h.period, h.value, t.start_value, t.target_value
		FROM task_progress_histories h
		JOIN tasks t ON t.id = h.task_id AND t.deleted_at IS NULL
		WHERE t.project_id = ? AND h.deleted_at IS NULL
		ORDER BY h.period ASC
	`, projectID).Scan(&histRows)

	// Fetch all tasks for the project (needed for current-month snapshot)
	type TaskRow struct {
		ID           uint
		StartValue   *float64
		TargetValue  float64
		CurrentValue float64
	}
	var taskRows []TaskRow
	dao.Database.Raw(`
		SELECT id, start_value, target_value, current_value
		FROM tasks
		WHERE project_id = ? AND deleted_at IS NULL
	`, projectID).Scan(&taskRows)

	// PeriodData accumulates sum of progress % and count for averaging
	type PeriodData struct {
		Period      string  `json:"period"`
		SumProgress float64 `json:"-"`
		Count       float64 `json:"-"`
		ExecPct     float64 `json:"exec_pct"`
		CumPlanned  float64 `json:"cum_planned"`
		CumAchieved float64 `json:"cum_achieved"`
		CumExecPct  float64 `json:"cum_exec_pct"`
		MsCount     int     `json:"ms_count"`
		MsDone      int     `json:"ms_done"`
	}

	// Build monthMap from recorded history
	monthMap := map[string]*PeriodData{}
	for _, r := range histRows {
		pd, exists := monthMap[r.Period]
		if !exists {
			pd = &PeriodData{Period: r.Period}
			monthMap[r.Period] = pd
		}
		sv := 0.0
		if r.StartValue != nil {
			sv = *r.StartValue
		}
		progress := milestoneTaskProgress(r.Value, sv, r.TargetValue)
		pd.SumProgress += progress
		pd.Count++
	}

	// Current month: ALWAYS use task.current_value as the live snapshot.
	// This overrides any history entry for the current month because milestones for the
	// current month may not yet be filled in (achieved_value still 0).
	currentMonthKey := time.Now().Format("2006-01")
	cur := &PeriodData{Period: currentMonthKey}
	for _, t := range taskRows {
		sv := 0.0
		if t.StartValue != nil {
			sv = *t.StartValue
		}
		progress := milestoneTaskProgress(t.CurrentValue, sv, t.TargetValue)
		cur.SumProgress += progress
		cur.Count++
	}
	monthMap[currentMonthKey] = cur

	// Generate ALL months from start_date to end_date (capped at current month)
	var startMonth, endMonth time.Time
	if project.StartDate != nil {
		startMonth = time.Date(project.StartDate.Year(), project.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else if len(histRows) > 0 {
		// parse earliest period from history
		t, _ := time.Parse("2006-01", histRows[0].Period)
		startMonth = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		c.JSON(http.StatusOK, gin.H{"periods": []interface{}{}, "start_date": project.StartDate, "end_date": project.EndDate})
		return
	}
	if project.EndDate != nil {
		endMonth = time.Date(project.EndDate.Year(), project.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		endMonth = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	now := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	if now.Before(endMonth) {
		endMonth = now
	}

	var monthKeys []string
	for m := startMonth; !m.After(endMonth); m = m.AddDate(0, 1, 0) {
		monthKeys = append(monthKeys, m.Format("2006-01"))
	}

	if len(monthKeys) == 0 {
		c.JSON(http.StatusOK, gin.H{"periods": []interface{}{}, "start_date": project.StartDate, "end_date": project.EndDate})
		return
	}

	// Compute per-period and cumulative execution %
	periods := make([]PeriodData, 0, len(monthKeys))
	var cumSum, cumCount float64

	for _, key := range monthKeys {
		pd, exists := monthMap[key]
		if !exists {
			pd = &PeriodData{Period: key}
		}
		// Period exec_pct = average progress % of milestones due this month
		if pd.Count > 0 {
			pd.ExecPct = math.Round(pd.SumProgress/pd.Count*10) / 10
		}
		cumSum += pd.SumProgress
		cumCount += pd.Count
		pd.CumPlanned = cumCount
		pd.CumAchieved = cumSum
		if cumCount > 0 {
			pd.CumExecPct = math.Round(cumSum/cumCount*10) / 10
		}
		pd.Period = key
		periods = append(periods, *pd)
	}

	c.JSON(http.StatusOK, gin.H{
		"periods":    periods,
		"start_date": project.StartDate,
		"end_date":   project.EndDate,
	})
}
