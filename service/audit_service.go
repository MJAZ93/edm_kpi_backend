package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type AuditService struct {
	Route      string
	Controller controller.AuditController
}

func (s AuditService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.List)
}
