package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type DashboardService struct {
	Route      string
	Controller controller.DashboardController
}

func (s DashboardService) Summary(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Summary)
}

func (s DashboardService) Performance(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Performance)
}

func (s DashboardService) DrillDown(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.DrillDown)
}

func (s DashboardService) MapData(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.MapData)
}

func (s DashboardService) Forecast(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Forecast)
}

func (s DashboardService) TopPerformers(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.TopPerformers)
}

func (s DashboardService) Timeline(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Timeline)
}

func (s DashboardService) Distribution(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Distribution)
}

func (s DashboardService) Benchmark(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Benchmark)
}
