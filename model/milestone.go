package model

import (
	"time"

	"gorm.io/gorm"
)

type Milestone struct {
	gorm.Model
	TaskID        uint       `gorm:"not null" json:"task_id"`
	Task          *Task      `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Title         string     `gorm:"not null;size:500" json:"title"`
	Description   string     `gorm:"type:text" json:"description,omitempty"`
	ScopeType     string     `gorm:"size:20" json:"scope_type,omitempty"` // NACIONAL, REGIONAL, ASC
	ScopeID       *uint      `json:"scope_id,omitempty"`
	Frequency     string     `gorm:"size:20" json:"frequency,omitempty"` // DAILY, WEEKLY, MONTHLY, QUARTERLY, BIANNUAL, ANNUAL
	PlannedValue  float64    `gorm:"type:decimal(15,2);not null" json:"planned_value"`
	AchievedValue   float64    `gorm:"type:decimal(15,2);default:0" json:"achieved_value"`
	AggregationType string     `gorm:"size:20;default:SUM_UP" json:"aggregation_type"` // SUM_UP, AVG, LAST, MANUAL
	PlannedDate   time.Time  `gorm:"not null" json:"planned_date"`
	AchievedDate  *time.Time `json:"achieved_date,omitempty"`
	PhotoURL      string     `gorm:"size:1000" json:"photo_url,omitempty"`
	Status        string     `gorm:"size:20;default:PENDING" json:"status"` // PENDING, IN_PROGRESS, DONE, BLOCKED
	Notes         string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedBy     uint       `gorm:"not null" json:"created_by"`
	Creator       *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	UpdatedBy     *uint      `json:"updated_by,omitempty"`
	Updater       *User      `gorm:"foreignKey:UpdatedBy" json:"updater,omitempty"`
	AssignedTo    *uint      `json:"assigned_to,omitempty"`
	Assignee      *User      `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
}
