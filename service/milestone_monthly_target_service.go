package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type MilestoneMonthlyTargetService struct {
	Controller controller.MilestoneMonthlyTargetController
}

// Register wires all routes for milestone monthly targets.
// Called from app/route.go.
func (s MilestoneMonthlyTargetService) Register(r *gin.RouterGroup) {
	r.GET("/milestones/:id/monthly-targets", s.Controller.ListByMilestone)
	r.PUT("/milestones/:id/monthly-targets", s.Controller.UpsertRow)
	r.PUT("/milestones/:id/monthly-targets/bulk", s.Controller.BulkUpsert)
	r.GET("/tasks/:id/monthly-chart", s.Controller.MonthlyChartForTask)
	r.GET("/projects/:id/monthly-chart", s.Controller.MonthlyChartForProject)
}
