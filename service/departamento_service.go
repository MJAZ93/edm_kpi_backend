package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type DepartamentoService struct {
	Route      string
	Controller controller.DepartamentoController
}

func (s DepartamentoService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.List)
}

func (s DepartamentoService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.Create)
}

func (s DepartamentoService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", s.Controller.Single)
}

func (s DepartamentoService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.Update)
}

func (s DepartamentoService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Delete)
}

func (s DepartamentoService) AddUser(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route+"/:id/users", middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.AddUser)
}

func (s DepartamentoService) RemoveUser(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id/users/:user_id", middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.RemoveUser)
}
