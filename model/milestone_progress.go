package model

import "gorm.io/gorm"

// MilestoneProgress records an incremental progress event on a milestone.
// Each "actualizar" call creates one row; the sum of all rows equals
// the milestone's current achieved_value.
type MilestoneProgress struct {
	gorm.Model
	MilestoneID    uint    `gorm:"not null;index" json:"milestone_id"`
	UserID         uint    `gorm:"not null" json:"user_id"`
	User           *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IncrementValue float64 `gorm:"type:decimal(15,2);not null" json:"increment_value"`
	Notes          string  `gorm:"type:text" json:"notes,omitempty"`
	PhotoURL       string  `gorm:"size:1000" json:"photo_url,omitempty"`
}
