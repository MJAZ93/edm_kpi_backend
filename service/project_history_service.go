package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type ProjectHistoryService struct {
	Route      string
	Controller controller.ProjectHistoryController
}

func (s ProjectHistoryService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route+"/:id/history", s.Controller.Create)
}

func (s ProjectHistoryService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id/history", s.Controller.List)
}

func (s ProjectHistoryService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/history/:entry_id", s.Controller.Update)
}

func (s ProjectHistoryService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/history/:entry_id", s.Controller.Delete)
}
