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

func (s DashboardService) ScopeStats(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.ScopeStats)
}

func (s DashboardService) EmployeeRanking(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.EmployeeRanking)
}

func (s DashboardService) DirecaoOverview(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.DirecaoOverview)
}

func (s DashboardService) DepartamentoOverview(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.DepartamentoOverview)
}

func (s DashboardService) MemberOverview(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.MemberOverview)
}

func (s DashboardService) RegionalOverview(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.RegionalOverview)
}
