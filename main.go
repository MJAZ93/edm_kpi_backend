package main

import (
	"log"

	"kpi-backend/app"
)

// @title           KPI Platform API
// @version         1.0
// @description     Backend for KPI management platform — project/task/milestone tracking with performance scoring
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	app.LoadEnv()

	if err := app.LoadDatabase(); err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	app.ServeApplication()
}
