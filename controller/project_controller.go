package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type ProjectController struct{}

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
		c.JSON(http.StatusOK, util.NewPaginatedResponse(list, int64(len(list)), params))
		return
	}

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	projectDao := dao.ProjectDao{}
	list, total, err := projectDao.ListScoped(params.Page, params.Limit, filters, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type ProjectInput struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	CreatorType  string  `json:"creator_type" binding:"required,oneof=ADMIN CA PELOURO DIRECAO DEPARTAMENTO"`
	CreatorOrgID *uint   `json:"creator_org_id"`
	ParentID     *uint   `json:"parent_id"`
	Weight       float64 `json:"weight"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	DirecaoIDs   []uint  `json:"direcao_ids"`
}

func (ProjectController) Create(c *gin.Context) {
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

	// Associate direcoes (for CA / PELOURO level projects)
	if len(input.DirecaoIDs) > 0 {
		if err := projectDao.SetDirecoes(project.ID, input.DirecaoIDs); err != nil {
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

func (ProjectController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
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
	projectDao := dao.ProjectDao{}
	project, err := projectDao.GetTree(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, project)
}
