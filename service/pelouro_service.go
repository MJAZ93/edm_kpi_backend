package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type PelouroService struct {
	Route      string
	Controller controller.PelouroController
}

func (s PelouroService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, middleware.RoleMiddleware("CA", "PELOURO"), s.Controller.List)
}

func (s PelouroService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, middleware.RoleMiddleware("CA"), s.Controller.Create)
}

func (s PelouroService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", middleware.RoleMiddleware("CA", "PELOURO"), s.Controller.Single)
}

func (s PelouroService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Update)
}

func (s PelouroService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Delete)
}
