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

	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	auditDao := dao.AuditDao{}
	list, total, err := auditDao.ListScoped(params.Page, params.Limit, filters, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}
