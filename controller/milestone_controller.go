package controller

import (
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
	PlannedValue float64 `json:"planned_value" binding:"required"`
	PlannedDate  string  `json:"planned_date" binding:"required"`
	Notes        string  `json:"notes"`
}

func (MilestoneController) Create(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Query("task_id"))

	var input MilestoneInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	plannedDate, err := parseDate(input.PlannedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid planned_date format"})
		return
	}

	milestone := model.Milestone{
		TaskID:       uint(taskID),
		Title:        input.Title,
		Description:  input.Description,
		ScopeType:    input.ScopeType,
		ScopeID:      input.ScopeID,
		PlannedValue: input.PlannedValue,
		PlannedDate:  *plannedDate,
		Notes:        input.Notes,
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
	PlannedValue  *float64 `json:"planned_value"`
	PlannedDate   *string  `json:"planned_date"`
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
	if input.PlannedValue != nil {
		milestone.PlannedValue = *input.PlannedValue
	}
	if input.PlannedDate != nil {
		t, _ := parseDate(*input.PlannedDate)
		if t != nil {
			milestone.PlannedDate = *t
		}
	}

	userID := util.ExtractUserID(c)
	milestone.UpdatedBy = &userID

	if err := milestoneDao.Update(&milestone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Recalculate parent task current_value
	taskDao := dao.TaskDao{}
	taskDao.RecalcCurrentValue(milestone.TaskID)

	// Refresh performance cache
	perfDao := dao.PerformanceDao{}
	go perfDao.RefreshForTask(milestone.TaskID)

	// Notify up the chain
	task, _ := taskDao.GetByID(milestone.TaskID)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "photo file required"})
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
