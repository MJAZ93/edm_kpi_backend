package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type PelouroController struct{}

func (PelouroController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	pelouroDao := dao.PelouroDao{}
	list, total, err := pelouroDao.ListScoped(params.Page, params.Limit, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type PelouroInput struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	ResponsibleID *uint  `json:"responsible_id"`
}

func (PelouroController) Create(c *gin.Context) {
	var input PelouroInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	pelouro := model.Pelouro{
		Name:          input.Name,
		Description:   input.Description,
		ResponsibleID: input.ResponsibleID,
		CreatedBy:     util.ExtractUserID(c),
	}

	pelouroDao := dao.PelouroDao{}
	if err := pelouroDao.Create(&pelouro); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("pelouro", pelouro.ID, util.ExtractUserID(c), "CREATE", nil, pelouro, c.ClientIP())

	c.JSON(http.StatusCreated, pelouro)
}

func (PelouroController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	pelouroDao := dao.PelouroDao{}
	pelouro, err := pelouroDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, pelouro)
}

func (PelouroController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	pelouroDao := dao.PelouroDao{}
	pelouro, err := pelouroDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := pelouro

	var input PelouroInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	pelouro.Name = input.Name
	pelouro.Description = input.Description
	pelouro.ResponsibleID = input.ResponsibleID
	pelouro.Responsible = nil

	if err := pelouroDao.Update(&pelouro); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("pelouro", pelouro.ID, util.ExtractUserID(c), "UPDATE", oldData, pelouro, c.ClientIP())

	c.JSON(http.StatusOK, pelouro)
}

func (PelouroController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	pelouroDao := dao.PelouroDao{}

	pelouro, err := pelouroDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := pelouroDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("pelouro", uint(id), util.ExtractUserID(c), "DELETE", pelouro, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
