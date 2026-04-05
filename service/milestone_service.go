package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type MilestoneService struct {
	Route      string
	Controller controller.MilestoneController
}

func (s MilestoneService) ListByTask(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.ListByTask)
}

func (s MilestoneService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, s.Controller.Create)
}

func (s MilestoneService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", s.Controller.Single)
}

func (s MilestoneService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", s.Controller.Update)
}

func (s MilestoneService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", s.Controller.Delete)
}

func (s MilestoneService) UploadPhoto(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route+"/:id/photo", s.Controller.UploadPhoto)
}
