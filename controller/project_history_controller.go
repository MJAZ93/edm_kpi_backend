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

// ExecutionHistory returns per-period execution percentages for a project.
// It groups all milestones (from all tasks in the project) by their planned_date
// month, computes achieved/planned ratio per period, and also returns cumulative %.
func (ProjectHistoryController) ExecutionHistory(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))

	// Get project for date range
	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// Fetch all milestones for tasks in this project
	type MsRow struct {
		PlannedDate   time.Time
		PlannedValue  float64
		AchievedValue float64
		Status        string
	}
	var rows []MsRow
	dao.Database.Raw(`
		SELECT m.planned_date, m.planned_value, m.achieved_value, m.status
		FROM milestones m
		JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
		WHERE t.project_id = ? AND m.deleted_at IS NULL
		ORDER BY m.planned_date ASC
	`, projectID).Scan(&rows)

	// Group by month (YYYY-MM)
	type PeriodData struct {
		Period        string  `json:"period"`
		Planned       float64 `json:"planned"`
		Achieved      float64 `json:"achieved"`
		ExecPct       float64 `json:"exec_pct"`
		CumPlanned    float64 `json:"cum_planned"`
		CumAchieved   float64 `json:"cum_achieved"`
		CumExecPct    float64 `json:"cum_exec_pct"`
		MsCount       int     `json:"ms_count"`
		MsDone        int     `json:"ms_done"`
	}

	// Build milestone data indexed by month
	monthMap := map[string]*PeriodData{}
	for _, r := range rows {
		key := r.PlannedDate.Format("2006-01")
		pd, exists := monthMap[key]
		if !exists {
			pd = &PeriodData{Period: key}
			monthMap[key] = pd
		}
		pd.Planned += r.PlannedValue
		pd.Achieved += r.AchievedValue
		pd.MsCount++
		if r.Status == "DONE" {
			pd.MsDone++
		}
	}

	// Generate ALL months from start_date to end_date
	var startMonth, endMonth time.Time
	if project.StartDate != nil {
		startMonth = time.Date(project.StartDate.Year(), project.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else if len(rows) > 0 {
		startMonth = time.Date(rows[0].PlannedDate.Year(), rows[0].PlannedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		c.JSON(http.StatusOK, gin.H{"periods": []interface{}{}, "start_date": project.StartDate, "end_date": project.EndDate})
		return
	}
	if project.EndDate != nil {
		endMonth = time.Date(project.EndDate.Year(), project.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		endMonth = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	// Also include current month if between start and end
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
	var cumPlanned, cumAchieved float64

	for _, key := range monthKeys {
		pd, exists := monthMap[key]
		if !exists {
			pd = &PeriodData{Period: key}
		}
		if pd.Planned > 0 {
			pd.ExecPct = math.Round((pd.Achieved/pd.Planned)*1000) / 10
		}
		cumPlanned += pd.Planned
		cumAchieved += pd.Achieved
		pd.CumPlanned = cumPlanned
		pd.CumAchieved = cumAchieved
		if cumPlanned > 0 {
			pd.CumExecPct = math.Round((cumAchieved/cumPlanned)*1000) / 10
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
