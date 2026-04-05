package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type NotificationService struct {
	Route      string
	Controller controller.NotificationController
}

func (s NotificationService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.List)
}

func (s NotificationService) MarkRead(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id/read", s.Controller.MarkRead)
}

func (s NotificationService) MarkAllRead(r *gin.RouterGroup, route string) {
	r.PUT("/"+s.Route+"/"+route, s.Controller.MarkAllRead)
}
