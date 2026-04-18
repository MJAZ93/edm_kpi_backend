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

	// Ensure milestones.aggregation_type defaults to SUM_UP and backfill NULLs
	Database.Exec("ALTER TABLE milestones ALTER COLUMN aggregation_type SET DEFAULT 'SUM_UP';")
	Database.Exec("UPDATE milestones SET aggregation_type = 'SUM_UP' WHERE aggregation_type IS NULL OR aggregation_type = '';")

	// Backfill monthly target rows for existing milestones. This is idempotent:
	// creates one row per (milestone, YYYY-MM) missing in the task's range.
	backfillMonthlyTargets()
}

// backfillMonthlyTargets ensures every existing milestone has one monthly row
// per month in its parent task's [start_date, end_date] range, and syncs
// achieved_value from the aggregated MilestoneProgress events per period.
// Safe to run on every boot — the INSERT is idempotent via ON CONFLICT, and
// the UPDATE always sets achieved_value to the canonical server-derived value.
func backfillMonthlyTargets() {
	// 1. Create missing monthly rows for every milestone in its task's range.
	Database.Exec(`
		INSERT INTO milestone_monthly_targets (
			milestone_id, period, planned_value, achieved_value,
			created_by, created_at, updated_at
		)
		SELECT
			m.id,
			to_char(gs, 'YYYY-MM') AS period,
			0, 0, COALESCE(m.created_by, 0), NOW(), NOW()
		FROM milestones m
		JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
		CROSS JOIN LATERAL generate_series(
			date_trunc('month', t.start_date),
			date_trunc('month', t.end_date),
			interval '1 month'
		) AS gs
		WHERE m.deleted_at IS NULL
			AND t.start_date IS NOT NULL
			AND t.end_date IS NOT NULL
		ON CONFLICT DO NOTHING;
	`)

	// 2. Sync achieved_value from MilestoneProgress.
	//    Each progress event's period_reference is converted to YYYY-MM using
	//    the same rules as periodRefToMonth in Go:
	//      YYYY-MM  → direct
	//      YYYY-Www → month of that ISO week's Monday
	//      YYYY-Qn  → first month of the quarter
	//      YYYY-Sn  → first month of the semester
	//      YYYY     → January of that year
	//      else     → created_at month
	//    Only the monthly row whose period matches the converted value is updated.
	Database.Exec(`
		UPDATE milestone_monthly_targets t
		SET achieved_value = COALESCE((
			SELECT SUM(mp.increment_value)
			FROM milestone_progresses mp
			WHERE mp.milestone_id = t.milestone_id
			  AND (
			    CASE
			      WHEN mp.period_reference ~ '^[0-9]{4}-[0-9]{2}$' THEN mp.period_reference
			      WHEN mp.period_reference ~ '^[0-9]{4}-W[0-9]{2}$' THEN
			        to_char(to_date(mp.period_reference || '-1', 'IYYY-"W"IW-ID'), 'YYYY-MM')
			      WHEN mp.period_reference ~ '^[0-9]{4}-Q[1-4]$' THEN
			        substring(mp.period_reference, 1, 4) || '-' ||
			        lpad((((substring(mp.period_reference, 7, 1)::int - 1) * 3 + 1)::text), 2, '0')
			      WHEN mp.period_reference ~ '^[0-9]{4}-S[12]$' THEN
			        substring(mp.period_reference, 1, 4) || '-' ||
			        CASE substring(mp.period_reference, 7, 1) WHEN '1' THEN '01' ELSE '07' END
			      WHEN mp.period_reference ~ '^[0-9]{4}$' THEN mp.period_reference || '-01'
			      ELSE to_char(mp.created_at AT TIME ZONE 'UTC', 'YYYY-MM')
			    END
			  ) = t.period
			  AND mp.deleted_at IS NULL
		), 0),
		updated_at = NOW()
		WHERE t.deleted_at IS NULL;
	`)

	// 3. Fallback: if a milestone has achieved_value > 0 but NO progress
	//    events (legacy data entered directly on the milestone), allocate
	//    the entire achieved amount to the milestone's planned_date month.
	//    Only touches rows where the backfilled achieved is still 0.
	Database.Exec(`
		UPDATE milestone_monthly_targets t
		SET achieved_value = m.achieved_value, updated_at = NOW()
		FROM milestones m
		WHERE m.id = t.milestone_id
		  AND m.deleted_at IS NULL
		  AND m.achieved_value > 0
		  AND t.achieved_value = 0
		  AND t.period = to_char(m.planned_date AT TIME ZONE 'UTC', 'YYYY-MM')
		  AND NOT EXISTS (
			SELECT 1 FROM milestone_progresses mp
			WHERE mp.milestone_id = m.id AND mp.deleted_at IS NULL
		  );
	`)
}
