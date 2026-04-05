package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type DirecaoController struct{}

func (DirecaoController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	direcaoDao := dao.DirecaoDao{}
	list, total, err := direcaoDao.List(params.Page, params.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type DirecaoInput struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	PelouroID     uint   `json:"pelouro_id" binding:"required"`
	ResponsibleID *uint  `json:"responsible_id"`
}

func (DirecaoController) Create(c *gin.Context) {
	var input DirecaoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	direcao := model.Direcao{
		Name:          input.Name,
		Description:   input.Description,
		PelouroID:     input.PelouroID,
		ResponsibleID: input.ResponsibleID,
		CreatedBy:     util.ExtractUserID(c),
	}

	direcaoDao := dao.DirecaoDao{}
	if err := direcaoDao.Create(&direcao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("direcao", direcao.ID, util.ExtractUserID(c), "CREATE", nil, direcao, c.ClientIP())

	c.JSON(http.StatusCreated, direcao)
}

func (DirecaoController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	direcaoDao := dao.DirecaoDao{}
	direcao, err := direcaoDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, direcao)
}

func (DirecaoController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	direcaoDao := dao.DirecaoDao{}
	direcao, err := direcaoDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := direcao

	var input DirecaoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	direcao.Name = input.Name
	direcao.Description = input.Description
	direcao.PelouroID = input.PelouroID
	direcao.ResponsibleID = input.ResponsibleID

	if err := direcaoDao.Update(&direcao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("direcao", direcao.ID, util.ExtractUserID(c), "UPDATE", oldData, direcao, c.ClientIP())

	c.JSON(http.StatusOK, direcao)
}

func (DirecaoController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	direcaoDao := dao.DirecaoDao{}

	direcao, err := direcaoDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := direcaoDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("direcao", uint(id), util.ExtractUserID(c), "DELETE", direcao, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
