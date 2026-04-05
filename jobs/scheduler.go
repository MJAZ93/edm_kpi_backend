package jobs

import (
	"log"
	"time"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"
)

func StartScheduler() {
	go blockerSLAJob()
	go performanceCacheJob()
	go forecastAlertJob()
	go milestoneOverdueJob()
	log.Println("[SCHEDULER] All background jobs started")
}

// Runs every hour: auto-approve expired blockers
func blockerSLAJob() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		blockerDao := dao.BlockerDao{}
		auditDao := dao.AuditDao{}
		notifDao := dao.NotificationDao{}

		expired, err := blockerDao.ListPendingExpired()
		if err != nil {
			log.Printf("[BLOCKER-SLA] Error: %v", err)
			continue
		}

		for _, b := range expired {
			if err := blockerDao.AutoApprove(b.ID); err != nil {
				log.Printf("[BLOCKER-SLA] Auto-approve failed for blocker %d: %v", b.ID, err)
				continue
			}

			auditDao.Write("blocker", b.ID, 0, "UPDATE", b, map[string]string{"status": "AUTO_APPROVED"}, "system")

			// If milestone blocker, set milestone to BLOCKED
			if b.EntityType == "MILESTONE" {
				milestoneDao := dao.MilestoneDao{}
				ms, err := milestoneDao.GetByID(b.EntityID)
				if err == nil {
					ms.Status = "BLOCKED"
					milestoneDao.Update(&ms)
					taskDao := dao.TaskDao{}
					taskDao.RecalcCurrentValue(ms.TaskID)
				}
			}

			// Notify reporter
			userDao := dao.UserDao{}
			reporter, err := userDao.GetByID(b.ReportedBy)
			if err == nil {
				notifDao.CreateAndEmail(reporter.ID, "Impedimento auto-aprovado",
					"O impedimento que reportou foi automaticamente aprovado (SLA expirado)",
					"BLOCKER_RESOLVED", "blocker", &b.ID)
				go util.EmailBlockerResolved(reporter.Email, reporter.Name, "tarefa", "AUTO-APROVADO")
			}

			log.Printf("[BLOCKER-SLA] Auto-approved blocker %d", b.ID)
		}
	}
}

// Runs nightly at midnight: refresh all performance scores
func performanceCacheJob() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(next.Sub(now))

		log.Println("[PERF-CACHE] Starting nightly refresh...")
		refreshAllPerformanceScores()
		log.Println("[PERF-CACHE] Refresh complete")
	}
}

func refreshAllPerformanceScores() {
	taskDao := dao.TaskDao{}
	perfDao := dao.PerformanceDao{}

	tasks, err := taskDao.ListActive()
	if err != nil {
		log.Printf("[PERF-CACHE] Error listing tasks: %v", err)
		return
	}

	for _, t := range tasks {
		if err := perfDao.RefreshForTask(t.ID); err != nil {
			log.Printf("[PERF-CACHE] Error refreshing task %d: %v", t.ID, err)
		}
	}
}

// Runs daily at 7 AM: check forecast alerts
func forecastAlertJob() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 7, 0, 0, 0, now.Location())
		if now.Hour() < 7 {
			next = time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
		}
		time.Sleep(next.Sub(now))

		log.Println("[FORECAST] Starting daily forecast check...")
		checkForecasts()
		log.Println("[FORECAST] Check complete")
	}
}

func checkForecasts() {
	taskDao := dao.TaskDao{}
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	tasks, err := taskDao.ListActive()
	if err != nil {
		return
	}

	for _, t := range tasks {
		if t.StartDate == nil || t.EndDate == nil {
			continue
		}
		if t.EndDate.Before(time.Now()) {
			continue
		}

		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}

		result := util.ForecastTask(t.ID, t.Title, startVal, t.TargetValue, t.CurrentValue, *t.StartDate, *t.EndDate)
		if result.Alert != nil {
			// Notify CA users
			caUsers, _ := userDao.GetByRole("CA")
			for _, ca := range caUsers {
				notifDao.CreateAndEmail(ca.ID, "Previsão de risco: "+t.Title,
					*result.AlertMessage, "FORECAST_RISK", "task", &t.ID)
				go util.EmailForecastRisk(ca.Email, ca.Name, t.Title, result.ProjectedFinalValue, t.TargetValue)
			}
		}
	}
}

// Runs daily at 8 AM: check overdue milestones
func milestoneOverdueJob() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 8, 0, 0, 0, now.Location())
		if now.Hour() < 8 {
			next = time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
		}
		time.Sleep(next.Sub(now))

		log.Println("[OVERDUE] Starting overdue milestone check...")
		checkOverdueMilestones()
		log.Println("[OVERDUE] Check complete")
	}
}

func checkOverdueMilestones() {
	milestoneDao := dao.MilestoneDao{}
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	overdue, err := milestoneDao.GetOverdue()
	if err != nil {
		return
	}

	for _, m := range overdue {
		// Notify the creator of the milestone
		creator, err := userDao.GetByID(m.CreatedBy)
		if err != nil {
			continue
		}

		var taskTitle string
		if m.Task != nil {
			taskTitle = m.Task.Title
		}

		notifDao.CreateAndEmail(creator.ID, "Milestone em atraso: "+m.Title,
			"O milestone '"+m.Title+"' está em atraso", "MILESTONE_OVERDUE", "milestone", &m.ID)
		go util.EmailMilestoneOverdue(creator.Email, creator.Name, m.Title, taskTitle)

		// Also notify CA
		caUsers, _ := userDao.GetByRole("CA")
		for _, ca := range caUsers {
			notifDao.CreateAndEmail(ca.ID, "Milestone em atraso: "+m.Title,
				"O milestone '"+m.Title+"' está em atraso", "MILESTONE_OVERDUE", "milestone", &m.ID)
		}
	}

	// Update overdue milestones to keep track
	for _, m := range overdue {
		m.Status = "IN_PROGRESS" // At least mark them as not just PENDING
		milestoneDao.Update(&m)
	}
}

// RefreshAllNow can be called manually for testing
func RefreshAllNow() {
	refreshAllPerformanceScores()
}

// SeedInitialAdmin creates an admin user if none exists
func SeedInitialAdmin() {
	userDao := dao.UserDao{}
	users, _ := userDao.GetByRole("CA")
	if len(users) > 0 {
		return
	}

	admin := &model.User{
		Name:     "Administrador",
		Email:    "admin@kpi.local",
		Password: "admin123",
		Role:     "CA",
		Active:   true,
	}
	if err := userDao.Create(admin); err != nil {
		log.Printf("[SEED] Failed to create admin: %v", err)
		return
	}
	log.Printf("[SEED] Created initial admin user: admin@kpi.local / admin123")
}
