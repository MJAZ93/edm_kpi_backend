package model

import "time"

type PerformanceCache struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	EntityType        string    `gorm:"not null;size:30;uniqueIndex:idx_perf_entity_period" json:"entity_type"` // CA, PELOURO, DIRECAO, DEPARTAMENTO, REGIAO, ASC, USER
	EntityID          uint      `gorm:"not null;uniqueIndex:idx_perf_entity_period" json:"entity_id"`
	Period            time.Time `gorm:"not null;uniqueIndex:idx_perf_entity_period" json:"period"` // first day of month
	ExecutionScore    float64   `gorm:"type:decimal(10,2)" json:"execution_score"`
	GoalScore         float64   `gorm:"type:decimal(10,2)" json:"goal_score"`
	TotalScore        float64   `gorm:"type:decimal(10,2)" json:"total_score"`
	TrafficLight      string    `gorm:"size:10" json:"traffic_light"` // GREEN, YELLOW, RED
	TasksTotal        int       `gorm:"default:0" json:"tasks_total"`
	TasksCompleted    int       `gorm:"default:0" json:"tasks_completed"`
	MilestonesTotal   int       `gorm:"default:0" json:"milestones_total"`
	MilestonesDone    int       `gorm:"default:0" json:"milestones_done"`
	MilestonesBlocked int       `gorm:"default:0" json:"milestones_blocked"`
	ComputedAt        time.Time `json:"computed_at"`
}
