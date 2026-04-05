package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type OrgService struct {
	Route      string
	Controller controller.OrgController
}

func (s OrgService) Tree(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Tree)
}
