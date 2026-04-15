package controller

import (
	"net/http"
	"strconv"

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
