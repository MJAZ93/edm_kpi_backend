package controller

import (
	"net/http"

	"kpi-backend/dao"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type AuditController struct{}

func (AuditController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	filters := make(map[string]interface{})

	if et := c.Query("entity_type"); et != "" {
		filters["entity_type"] = et
	}
	if eid := c.Query("entity_id"); eid != "" {
		filters["entity_id"] = eid
	}

	auditDao := dao.AuditDao{}
	list, total, err := auditDao.List(params.Page, params.Limit, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}
