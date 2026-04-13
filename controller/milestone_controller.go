package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type MilestoneController struct{}

func (MilestoneController) ListByTask(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Query("task_id"))
	params := util.ParsePagination(c)

	// Verify the task is within user's scope
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	if !scope.IsGlobal {
		taskDao := dao.TaskDao{}
		task, err := taskDao.GetByID(uint(taskID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		if !scope.CanSeeTask(task) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "task outside your scope"})
			return
		}
	}

	milestoneDao := dao.MilestoneDao{}
	list, total, err := milestoneDao.ListByTask(uint(taskID), params.Page, params.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type MilestoneInput struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	ScopeType    string  `json:"scope_type"`
	ScopeID      *uint   `json:"scope_id"`
	Frequency    string  `json:"frequency"`
	PlannedValue float64 `json:"planned_value" binding:"required"`
	PlannedDate  string  `json:"planned_date" binding:"required"`
	Notes        string  `json:"notes"`
	AssignedTo   *uint   `json:"assigned_to"`
}

func normalizeFrequency(input, fallback string) (string, error) {
	freq := input
	if freq == "" {
		freq = fallback
	}

	switch freq {
	case "DAILY", "WEEKLY", "MONTHLY", "QUARTERLY", "BIANNUAL", "ANNUAL":
		return freq, nil
	case "":
		return "", fmt.Errorf("frequency is required")
	default:
		return "", fmt.Errorf("invalid frequency")
	}
}

func validateMilestoneAssignee(task model.Task, assignedTo *uint) error {
	if assignedTo == nil {
		return nil
	}
	if task.OwnerType != "DEPARTAMENTO" {
		return fmt.Errorf("assigned user requires a department-owned task")
	}

	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByID(task.OwnerID)
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

	return fmt.Errorf("assigned user must belong to the task department")
}

func (MilestoneController) Create(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Query("task_id"))

	var input MilestoneInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "task not found"})
		return
	}

	plannedDate, err := parseDate(input.PlannedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid planned_date format"})
		return
	}
	frequency, err := normalizeFrequency(input.Frequency, task.Frequency)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	if err := validateMilestoneAssignee(task, input.AssignedTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	milestone := model.Milestone{
		TaskID:       uint(taskID),
		Title:        input.Title,
		Description:  input.Description,
		ScopeType:    input.ScopeType,
		ScopeID:      input.ScopeID,
		Frequency:    frequency,
		PlannedValue: input.PlannedValue,
		PlannedDate:  *plannedDate,
		Notes:        input.Notes,
		AssignedTo:   input.AssignedTo,
		Status:       "PENDING",
		CreatedBy:    util.ExtractUserID(c),
	}

	milestoneDao := dao.MilestoneDao{}
	if err := milestoneDao.Create(&milestone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("milestone", milestone.ID, util.ExtractUserID(c), "CREATE", nil, milestone, c.ClientIP())

	c.JSON(http.StatusCreated, milestone)
}

func (MilestoneController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	milestoneDao := dao.MilestoneDao{}
	milestone, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, milestone)
}

type MilestoneUpdateInput struct {
	AchievedValue *float64 `json:"achieved_value"`
	AchievedDate  *string  `json:"achieved_date"`
	Status        *string  `json:"status"`
	Notes         *string  `json:"notes"`
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	ScopeType     *string  `json:"scope_type"`
	ScopeID       *uint    `json:"scope_id"`
	Frequency     *string  `json:"frequency"`
	PlannedValue  *float64 `json:"planned_value"`
	PlannedDate   *string  `json:"planned_date"`
	AssignedTo    *uint    `json:"assigned_to"`
}

func (MilestoneController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	milestoneDao := dao.MilestoneDao{}
	milestone, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := milestone

	var input MilestoneUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	taskDao := dao.TaskDao{}
	task, taskErr := taskDao.GetByID(milestone.TaskID)
	if taskErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "task not found"})
		return
	}

	if input.AchievedValue != nil {
		milestone.AchievedValue = *input.AchievedValue
	}
	if input.AchievedDate != nil {
		t, _ := parseDate(*input.AchievedDate)
		milestone.AchievedDate = t
	}
	if input.Status != nil {
		milestone.Status = *input.Status
	}
	if input.Notes != nil {
		milestone.Notes = *input.Notes
	}
	if input.Title != nil {
		milestone.Title = *input.Title
	}
	if input.Description != nil {
		milestone.Description = *input.Description
	}
	if input.ScopeType != nil {
		milestone.ScopeType = *input.ScopeType
	}
	if input.ScopeID != nil {
		milestone.ScopeID = input.ScopeID
	}
	if input.Frequency != nil {
		frequency, err := normalizeFrequency(*input.Frequency, milestone.Frequency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
			return
		}
		milestone.Frequency = frequency
	}
	if input.PlannedValue != nil {
		milestone.PlannedValue = *input.PlannedValue
	}
	if input.PlannedDate != nil {
		t, _ := parseDate(*input.PlannedDate)
		if t != nil {
			milestone.PlannedDate = *t
		}
	}
	if input.AssignedTo != nil {
		if err := validateMilestoneAssignee(task, input.AssignedTo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
			return
		}
		milestone.AssignedTo = input.AssignedTo
	}

	userID := util.ExtractUserID(c)
	milestone.UpdatedBy = &userID

	if err := milestoneDao.Update(&milestone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Recalculate parent task current_value
	taskDao.RecalcCurrentValue(milestone.TaskID)

	// Refresh performance cache: task owner + project-level
	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(milestone.TaskID)

	// Notify up the chain
	task, _ = taskDao.GetByID(milestone.TaskID)
	if task.ProjectID != 0 {
		go perfDao.RefreshForProject(task.ProjectID)
	}
	go notifyTaskUpdateChain(task, userID)

	auditDao := dao.AuditDao{}
	auditDao.Write("milestone", milestone.ID, userID, "UPDATE", oldData, milestone, c.ClientIP())

	result, _ := milestoneDao.GetByID(uint(id))
	c.JSON(http.StatusOK, result)
}

func (MilestoneController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	milestoneDao := dao.MilestoneDao{}

	milestone, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := milestoneDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Recalc after delete
	taskDao := dao.TaskDao{}
	taskDao.RecalcCurrentValue(milestone.TaskID)

	auditDao := dao.AuditDao{}
	auditDao.Write("milestone", uint(id), util.ExtractUserID(c), "DELETE", milestone, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (MilestoneController) UploadPhoto(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "attachment file required"})
		return
	}
	defer file.Close()

	if err := util.ValidatePhoto(header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	url, err := util.UploadPhoto(file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "upload failed"})
		return
	}

	milestoneDao := dao.MilestoneDao{}
	if err := milestoneDao.UpdatePhoto(uint(id), url); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"photo_url": url})
}

// AddProgress records an incremental progress event and accumulates into achieved_value.
func (MilestoneController) AddProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID := util.ExtractUserID(c)

	var input struct {
		IncrementValue  float64 `json:"increment_value" binding:"required,min=0.01"`
		PeriodReference string  `json:"period_reference"`
		Notes           string  `json:"notes"`
		Status          string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	milestoneDao := dao.MilestoneDao{}
	ms, err := milestoneDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// Validate uniqueness of period_reference per milestone (one entry per period)
	if input.PeriodReference != "" {
		var count int64
		dao.Database.Model(&model.MilestoneProgress{}).
			Where("milestone_id = ? AND period_reference = ? AND deleted_at IS NULL", uint(id), input.PeriodReference).
			Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "period_already_recorded", "message": "Já existe um registo para este período."})
			return
		}
	}

	// Record the progress event
	progress := model.MilestoneProgress{
		MilestoneID:     uint(id),
		UserID:          userID,
		IncrementValue:  input.IncrementValue,
		PeriodReference: input.PeriodReference,
		Notes:           input.Notes,
	}
	if err := dao.Database.Create(&progress).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Accumulate achieved_value
	ms.AchievedValue += input.IncrementValue
	ms.UpdatedBy = &userID

	// Detect reduction goal from parent task (target < start)
	taskDao := dao.TaskDao{}
	parentTask, _ := taskDao.GetByID(ms.TaskID)
	isReduction := false
	if parentTask.StartValue != nil && parentTask.TargetValue < *parentTask.StartValue {
		isReduction = true
	}

	// Update status if provided; otherwise auto-complete based on goal direction
	if input.Status != "" {
		ms.Status = input.Status
	} else if isReduction && ms.AchievedValue <= ms.PlannedValue {
		// Reduction goal: achieved at or below planned target = done
		ms.Status = "DONE"
	} else if !isReduction && ms.AchievedValue >= ms.PlannedValue {
		// Growth goal: achieved at or above planned target = done
		ms.Status = "DONE"
	}

	if err := milestoneDao.Update(&ms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Recalculate parent task current_value (sum of all milestone achieved_values)
	taskDao.RecalcCurrentValue(ms.TaskID)

	// Refresh performance cache asynchronously
	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(ms.TaskID)

	// Notify up the chain
	go notifyTaskUpdateChain(parentTask, userID)

	c.JSON(http.StatusOK, gin.H{
		"milestone":      ms,
		"progress_event": progress,
		"new_total":      ms.AchievedValue,
	})
}

// ListProgress returns all progress events for a milestone.
func (MilestoneController) ListProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var events []model.MilestoneProgress
	dao.Database.Preload("User").
		Where("milestone_id = ?", id).
		Order("created_at DESC").
		Find(&events)

	c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
}
