package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type DepartamentoController struct{}

func (DepartamentoController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	deptDao := dao.DepartamentoDao{}
	list, total, err := deptDao.ListScoped(params.Page, params.Limit, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type DepartamentoInput struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	DirecaoID     uint   `json:"direcao_id" binding:"required"`
	ResponsibleID *uint  `json:"responsible_id"`
}

func (DepartamentoController) Create(c *gin.Context) {
	var input DepartamentoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	dept := model.Departamento{
		Name:          input.Name,
		Description:   input.Description,
		DirecaoID:     input.DirecaoID,
		ResponsibleID: input.ResponsibleID,
		CreatedBy:     util.ExtractUserID(c),
	}

	deptDao := dao.DepartamentoDao{}
	if err := deptDao.Create(&dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("departamento", dept.ID, util.ExtractUserID(c), "CREATE", nil, dept, c.ClientIP())

	c.JSON(http.StatusCreated, dept)
}

func (DepartamentoController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, dept)
}

func (DepartamentoController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := dept

	var input DepartamentoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	dept.Name = input.Name
	dept.Description = input.Description
	dept.DirecaoID = input.DirecaoID
	dept.ResponsibleID = input.ResponsibleID
	dept.Responsible = nil
	dept.Direcao = nil

	if err := deptDao.Update(&dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("departamento", dept.ID, util.ExtractUserID(c), "UPDATE", oldData, dept, c.ClientIP())

	c.JSON(http.StatusOK, dept)
}

func (DepartamentoController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	deptDao := dao.DepartamentoDao{}

	dept, err := deptDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := deptDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("departamento", uint(id), util.ExtractUserID(c), "DELETE", dept, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

type AddUserInput struct {
	UserID uint `json:"user_id" binding:"required"`
}

func (DepartamentoController) AddUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input AddUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	deptDao := dao.DepartamentoDao{}
	if err := deptDao.AddUser(uint(id), input.UserID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": "user already in department"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user added"})
}

func (DepartamentoController) RemoveUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := strconv.Atoi(c.Param("user_id"))

	deptDao := dao.DepartamentoDao{}
	if err := deptDao.RemoveUser(uint(id), uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed"})
}
