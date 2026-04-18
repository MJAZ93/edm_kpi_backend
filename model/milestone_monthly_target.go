package model

import "gorm.io/gorm"

// MilestoneMonthlyTarget represents one month of planned (meta) and
// achieved (realizado) values for a milestone / indicator.
// One row per milestone per month (YYYY-MM), auto-generated when a
// milestone is created (covering the task's start_date..end_date range).
// This is the canonical representation for the Meta vs Realizado chart
// at milestone, task and project levels.
type MilestoneMonthlyTarget struct {
	gorm.Model
	MilestoneID   uint    `gorm:"not null;uniqueIndex:idx_mmt_ms_period" json:"milestone_id"`
	Period        string  `gorm:"not null;size:7;uniqueIndex:idx_mmt_ms_period" json:"period"` // "2026-04"
	PlannedValue  float64 `gorm:"type:decimal(15,2);default:0" json:"planned_value"`
	AchievedValue float64 `gorm:"type:decimal(15,2);default:0" json:"achieved_value"`
	Notes         string  `gorm:"type:text" json:"notes,omitempty"`
	CreatedBy     uint    `json:"created_by"`
	UpdatedBy     *uint   `json:"updated_by,omitempty"`
}
