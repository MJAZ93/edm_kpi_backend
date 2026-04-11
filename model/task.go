package model

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	ProjectID     uint        `gorm:"not null" json:"project_id"`
	Project       *Project    `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	ParentTaskID  *uint       `json:"parent_task_id,omitempty"`
	ParentTask    *Task       `gorm:"foreignKey:ParentTaskID" json:"parent_task,omitempty"`
	ChildTasks    []Task      `gorm:"foreignKey:ParentTaskID" json:"child_tasks,omitempty"`
	Title         string      `gorm:"not null;size:500" json:"title"`
	Description   string      `gorm:"type:text" json:"description,omitempty"`
	OwnerType     string      `gorm:"not null;size:20" json:"owner_type"` // DIRECAO, DEPARTAMENTO
	OwnerID       uint        `gorm:"not null" json:"owner_id"`           // direcao_id or departamento_id
	Frequency     string      `gorm:"not null;size:20" json:"frequency"`  // DAILY, WEEKLY, MONTHLY, QUARTERLY, BIANNUAL, ANNUAL
	GoalLabel     string      `gorm:"size:255" json:"goal_label,omitempty"`
	StartValue    *float64    `gorm:"type:decimal(15,2)" json:"start_value"`
	TargetValue   float64     `gorm:"type:decimal(15,2);not null" json:"target_value"`
	CurrentValue    float64     `gorm:"type:decimal(15,2);default:0" json:"current_value"`
	AggregationType string      `gorm:"size:20;default:SUM_UP" json:"aggregation_type"` // SUM_UP, SUM_DOWN, AVG
	Weight          float64     `gorm:"type:decimal(5,2);default:100.0" json:"weight"`
	StartDate     *time.Time  `json:"start_date,omitempty"`
	EndDate       *time.Time  `json:"end_date,omitempty"`
	NextUpdateDue *time.Time  `json:"next_update_due,omitempty"`
	Status        string      `gorm:"size:20;default:ACTIVE" json:"status"` // ACTIVE, COMPLETED, CANCELLED
	CreatedBy     uint        `gorm:"not null" json:"created_by"`
	Creator       *User       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	AssignedTo    *uint       `json:"assigned_to,omitempty"`
	Assignee      *User       `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	Scopes        []TaskScope `gorm:"foreignKey:TaskID" json:"scopes,omitempty"`
	Milestones    []Milestone `gorm:"foreignKey:TaskID" json:"milestones,omitempty"`
}

type TaskScope struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	TaskID    uint   `gorm:"not null" json:"task_id"`
	ScopeType string `gorm:"not null;size:20" json:"scope_type"` // NACIONAL, REGIONAL, ASC
	ScopeID   *uint  `json:"scope_id,omitempty"`                 // regiao_id or asc_id; nil for NACIONAL
}
