package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type DirecaoService struct {
	Route      string
	Controller controller.DirecaoController
}

func (s DirecaoService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.List)
}

func (s DirecaoService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, middleware.RoleMiddleware("CA", "PELOURO"), s.Controller.Create)
}

func (s DirecaoService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", middleware.RoleMiddleware("CA", "PELOURO", "DIRECAO"), s.Controller.Single)
}

func (s DirecaoService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", middleware.RoleMiddleware("CA", "PELOURO"), s.Controller.Update)
}

func (s DirecaoService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Delete)
}
