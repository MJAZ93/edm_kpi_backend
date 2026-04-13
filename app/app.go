package app

import (
	"fmt"
	"log"
	"os"

	"kpi-backend/dao"
	"kpi-backend/jobs"
	"kpi-backend/middleware"
	"kpi-backend/model"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Version is set at build time via -ldflags or defaults to "dev".
var Version = "1.3.0"

func LoadEnv() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("failed loading .env: ", err)
	}
}

func LoadDatabase() error {
	dao.Connect()
	dao.SetupExtensions()

	models := []interface{}{
		&model.User{},
		&model.Pelouro{},
		&model.Direcao{},
		&model.Departamento{},
		&model.DepartamentoUser{},
		&model.Regiao{},
		&model.ASC{},
		&model.Project{},
		&model.Task{},
		&model.TaskScope{},
		&model.Milestone{},
		&model.MilestoneProgress{},
		&model.Blocker{},
		&model.AuditLog{},
		&model.Notification{},
		&model.PerformanceCache{},
	}

	for _, m := range models {
		if err := dao.Database.AutoMigrate(m); err != nil {
			return err
		}
	}

	dao.SetupIndexes()
	dao.RunMigrations()
	return nil
}

func ServeApplication() {
	r := gin.Default()

	Swagger(r)

	r.Use(middleware.DefaultAuthMiddleware())

	base := r.Group("/api/v1")

	PublicRoutes(base.Group("/public"))

	priv := base.Group("/private")
	priv.Use(middleware.JWTAuthMiddleware())
	PrivateRoutes(priv)

	// Serve static uploads
	r.Static("/uploads", os.Getenv("UPLOAD_DIR"))

	// Seed admin user if none exists
	jobs.SeedInitialAdmin()

	// Start background jobs
	jobs.StartScheduler()

	// Seed performance cache on startup (non-blocking)
	go func() {
		jobs.RefreshAllNow()
	}()

	ip, port := os.Getenv("IP"), os.Getenv("PORT")
	log.Printf("Starting KPI Platform on %s:%s", ip, port)
	if err := r.Run(fmt.Sprintf("%s:%s", ip, port)); err != nil {
		log.Fatalf("gin failed to start: %v", err)
	}
}
