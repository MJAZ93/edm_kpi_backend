package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type ProjectService struct {
	Route      string
	Controller controller.ProjectController
}

func (s ProjectService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, s.Controller.List)
}

func (s ProjectService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, s.Controller.Create)
}

func (s ProjectService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", s.Controller.Single)
}

func (s ProjectService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", s.Controller.Update)
}

func (s ProjectService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", s.Controller.Delete)
}

func (s ProjectService) Tree(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id/tree", s.Controller.Tree)
}

func (s ProjectService) UpdateProgress(r *gin.RouterGroup, _ string) {
	r.PATCH("/"+s.Route+"/:id/progress", s.Controller.UpdateProgress)
}
