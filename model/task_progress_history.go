package model

import "gorm.io/gorm"

// TaskProgressHistory records the current_value of a task for a given month (YYYY-MM).
// One record per task per month — upserted whenever current_value changes.
// Used by the execution history chart to show monthly progress timelines.
type TaskProgressHistory struct {
	gorm.Model
	TaskID    uint    `gorm:"not null;uniqueIndex:idx_task_period" json:"task_id"`
	Period    string  `gorm:"not null;size:7;uniqueIndex:idx_task_period" json:"period"` // "2026-04"
	Value     float64 `gorm:"type:decimal(15,2)" json:"value"`
	CreatedBy uint    `json:"created_by"`
}
