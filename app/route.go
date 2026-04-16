package app

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"
	"kpi-backend/service"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func PublicRoutes(r *gin.RouterGroup) {
	authSvc := service.AuthService{Route: "auth", Controller: controller.AuthController{}}
	authSvc.Login(r, "login")
	authSvc.ForgotPassword(r, "forgot-password")
	authSvc.ResetPassword(r, "reset-password")

	r.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": Version})
	})
}

func PrivateRoutes(r *gin.RouterGroup) {
	// Auth (private)
	r.PUT("auth/change-password", controller.AuthController{}.ChangePassword)

	// Users
	userSvc := service.UserService{Route: "users", Controller: controller.UserController{}}
	userSvc.Me(r, "me")
	userSvc.List(r, "")
	userSvc.Create(r, "")
	userSvc.Single(r, "")
	userSvc.Update(r, "")
	userSvc.Delete(r, "")

	// Organisation - Pelouros
	pelouroSvc := service.PelouroService{Route: "pelouros", Controller: controller.PelouroController{}}
	pelouroSvc.List(r, "")
	pelouroSvc.Create(r, "")
	pelouroSvc.Single(r, "")
	pelouroSvc.Update(r, "")
	pelouroSvc.Delete(r, "")

	// Organisation - Direções
	direcaoSvc := service.DirecaoService{Route: "direcoes", Controller: controller.DirecaoController{}}
	direcaoSvc.List(r, "")
	direcaoSvc.Create(r, "")
	direcaoSvc.Single(r, "")
	direcaoSvc.Update(r, "")
	direcaoSvc.Delete(r, "")

	// Organisation - Departamentos
	deptSvc := service.DepartamentoService{Route: "departamentos", Controller: controller.DepartamentoController{}}
	deptSvc.List(r, "")
	deptSvc.Create(r, "")
	deptSvc.Single(r, "")
	deptSvc.Update(r, "")
	deptSvc.Delete(r, "")
	deptSvc.AddUser(r, "")
	deptSvc.RemoveUser(r, "")

	// Organisation Tree
	orgSvc := service.OrgService{Route: "org", Controller: controller.OrgController{}}
	orgSvc.Tree(r, "tree")

	// Geography - Regiões
	geoSvc := service.GeoService{Route: "geo", Controller: controller.GeoController{}}
	geoSvc.ListRegioes(r, "regioes")
	geoSvc.CreateRegiao(r, "regioes")
	geoSvc.SingleRegiao(r, "regioes")
	geoSvc.UpdateRegiao(r, "regioes")
	geoSvc.DeleteRegiao(r, "regioes")
	geoSvc.ListASCs(r, "ascs")
	geoSvc.CreateASC(r, "ascs")
	geoSvc.SingleASC(r, "ascs")
	geoSvc.UpdateASC(r, "ascs")
	geoSvc.DeleteASC(r, "ascs")

	// Projects
	projectSvc := service.ProjectService{Route: "projects", Controller: controller.ProjectController{}}
	projectSvc.List(r, "")
	projectSvc.Create(r, "")
	projectSvc.Single(r, "")
	projectSvc.Update(r, "")
	projectSvc.Delete(r, "")
	projectSvc.Tree(r, "")
	projectSvc.UpdateProgress(r, "")

	// Project History
	projectHistorySvc := service.ProjectHistoryService{Route: "projects", Controller: controller.ProjectHistoryController{}}
	projectHistorySvc.Create(r, "")
	projectHistorySvc.List(r, "")
	projectHistorySvc.Update(r, "")
	projectHistorySvc.Delete(r, "")
	projectHistorySvc.ExecutionHistory(r, "")

	// Tasks
	taskSvc := service.TaskService{Route: "tasks", Controller: controller.TaskController{}}
	taskSvc.ListByProject(r, "")
	taskSvc.Create(r, "")
	taskSvc.Single(r, "")
	taskSvc.Update(r, "")
	taskSvc.Delete(r, "")

	// Milestones
	milestoneSvc := service.MilestoneService{Route: "milestones", Controller: controller.MilestoneController{}}
	milestoneSvc.ListByTask(r, "")
	milestoneSvc.Create(r, "")
	milestoneSvc.Single(r, "")
	milestoneSvc.Update(r, "")
	milestoneSvc.Delete(r, "")
	milestoneSvc.UploadPhoto(r, "")
	milestoneSvc.AddProgress(r, "")
	milestoneSvc.ListProgress(r, "")
	milestoneSvc.UpdateProgress(r, "")

	// Blockers
	blockerSvc := service.BlockerService{Route: "blockers", Controller: controller.BlockerController{}}
	blockerSvc.List(r, "")
	blockerSvc.Create(r, "")
	blockerSvc.Single(r, "")
	blockerSvc.Approve(r, "")
	blockerSvc.Reject(r, "")

	// Dashboard
	dashSvc := service.DashboardService{Route: "dashboard", Controller: controller.DashboardController{}}
	dashSvc.Summary(r, "summary")
	dashSvc.Performance(r, "performance")
	dashSvc.DrillDown(r, "drill-down")
	dashSvc.MapData(r, "map")
	dashSvc.Forecast(r, "forecast")
	dashSvc.TopPerformers(r, "top-performers")
	dashSvc.Timeline(r, "timeline")
	dashSvc.Distribution(r, "distribution")
	dashSvc.Benchmark(r, "benchmark")
	dashSvc.ScopeStats(r, "scope-stats")
	dashSvc.EmployeeRanking(r, "employee-ranking")
	dashSvc.DirecaoOverview(r, "direcao-overview")
	dashSvc.DepartamentoOverview(r, "departamento-overview")
	dashSvc.MemberOverview(r, "member-overview")
	dashSvc.RegionalOverview(r, "regional-overview")
	dashSvc.DirecaoMilestones(r, "direcao-milestones")
	dashSvc.DepartamentoDetail(r, "departamento-detail")
	dashSvc.UserDetail(r, "user-detail")

	// Feedback
	feedbackSvc := service.FeedbackService{Route: "feedback", Controller: controller.FeedbackController{}}
	feedbackSvc.Create(r, "")
	feedbackSvc.Reply(r, "")
	feedbackSvc.ListReceived(r, "")
	feedbackSvc.ListSent(r, "")
	feedbackSvc.ListByTarget(r, "")
	feedbackSvc.UnreadCount(r, "")
	feedbackSvc.MarkAsRead(r, "")
	feedbackSvc.Delete(r, "")

	// Notifications
	notifSvc := service.NotificationService{Route: "notifications", Controller: controller.NotificationController{}}
	notifSvc.List(r, "")
	notifSvc.MarkRead(r, "")
	notifSvc.MarkAllRead(r, "read-all")

	// Audit
	auditSvc := service.AuditService{Route: "audit", Controller: controller.AuditController{}}
	auditSvc.List(r, "")

	// Role-restricted routes
	ca := r.Group("")
	ca.Use(middleware.RoleMiddleware("CA"))
	_ = ca // CA-only routes are handled via role checks inside controllers
}

func Swagger(r *gin.Engine) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
