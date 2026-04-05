package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type TaskService struct {
	Route      string
	Controller controller.TaskController
}

func (s TaskService) ListByProject(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.ListByProject)
}

func (s TaskService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, s.Controller.Create)
}

func (s TaskService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", s.Controller.Single)
}

func (s TaskService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", s.Controller.Update)
}

func (s TaskService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", s.Controller.Delete)
}
