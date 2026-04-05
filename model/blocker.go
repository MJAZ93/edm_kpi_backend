package model

import (
	"time"

	"gorm.io/gorm"
)

type Blocker struct {
	gorm.Model
	EntityType      string     `gorm:"not null;size:20" json:"entity_type"` // TASK, MILESTONE
	EntityID        uint       `gorm:"not null" json:"entity_id"`
	BlockerType     string     `gorm:"not null;size:20" json:"blocker_type"` // LOGISTIC, FINANCIAL, TECHNICAL, LEGAL
	Description     string     `gorm:"type:text;not null" json:"description"`
	ReportedBy      uint       `gorm:"not null" json:"reported_by"`
	Reporter        *User      `gorm:"foreignKey:ReportedBy" json:"reporter,omitempty"`
	ApprovedBy      *uint      `json:"approved_by,omitempty"`
	Approver        *User      `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	Status          string     `gorm:"size:20;default:PENDING" json:"status"` // PENDING, APPROVED, REJECTED, AUTO_APPROVED
	SLADays         int        `gorm:"default:3" json:"sla_days"`
	AutoApproveAt   *time.Time `json:"auto_approve_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	RejectionReason string     `gorm:"type:text" json:"rejection_reason,omitempty"`
}
