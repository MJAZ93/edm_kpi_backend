package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type GeoService struct {
	Route      string
	Controller controller.GeoController
}

func (s GeoService) ListRegioes(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.ListRegioes)
}

func (s GeoService) CreateRegiao(r *gin.RouterGroup, route string) {
	r.POST("/"+s.Route+"/"+route, middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.CreateRegiao)
}

func (s GeoService) SingleRegiao(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route+"/:id", s.Controller.SingleRegiao)
}

func (s GeoService) UpdateRegiao(r *gin.RouterGroup, route string) {
	r.PUT("/"+s.Route+"/"+route+"/:id", middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.UpdateRegiao)
}

func (s GeoService) DeleteRegiao(r *gin.RouterGroup, route string) {
	r.DELETE("/"+s.Route+"/"+route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.DeleteRegiao)
}

func (s GeoService) ListASCs(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.ListASCs)
}

func (s GeoService) CreateASC(r *gin.RouterGroup, route string) {
	r.POST("/"+s.Route+"/"+route, middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.CreateASC)
}

func (s GeoService) SingleASC(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route+"/:id", s.Controller.SingleASC)
}

func (s GeoService) UpdateASC(r *gin.RouterGroup, route string) {
	r.PUT("/"+s.Route+"/"+route+"/:id", middleware.RoleMiddleware("CA", "DIRECAO"), s.Controller.UpdateASC)
}

func (s GeoService) DeleteASC(r *gin.RouterGroup, route string) {
	r.DELETE("/"+s.Route+"/"+route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.DeleteASC)
}
