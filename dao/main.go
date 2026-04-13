package dao

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Database *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic(err)
	}

	Database = db
}

func SetupExtensions() {
	Database.Exec("CREATE EXTENSION IF NOT EXISTS postgis;")
}

func SetupIndexes() {
	Database.Exec("CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id);")
	Database.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, is_read);")
	Database.Exec("CREATE INDEX IF NOT EXISTS idx_performance_cache_entity ON performance_caches(entity_type, entity_id, period);")
}

// RunMigrations applies column-type changes that AutoMigrate cannot handle.
func RunMigrations() {
	// Widen score columns from decimal(5,2) to decimal(10,2) to support negative scores
	Database.Exec("ALTER TABLE performance_caches ALTER COLUMN execution_score TYPE decimal(10,2);")
	Database.Exec("ALTER TABLE performance_caches ALTER COLUMN goal_score TYPE decimal(10,2);")
	Database.Exec("ALTER TABLE performance_caches ALTER COLUMN total_score TYPE decimal(10,2);")
}
