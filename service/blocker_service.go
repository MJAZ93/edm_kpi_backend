package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type BlockerService struct {
	Route      string
	Controller controller.BlockerController
}

func (s BlockerService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.List)
}

func (s BlockerService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, s.Controller.Create)
}

func (s BlockerService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", s.Controller.Single)
}

func (s BlockerService) Approve(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id/approve", middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.Approve)
}

func (s BlockerService) Reject(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id/reject", middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.Reject)
}
