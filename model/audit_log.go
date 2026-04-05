package model

import (
	"time"

	"gorm.io/datatypes"
)

type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EntityType string         `gorm:"not null;size:50" json:"entity_type"`
	EntityID   uint           `gorm:"not null" json:"entity_id"`
	ChangedBy  uint           `gorm:"not null" json:"changed_by"`
	Changer    *User          `gorm:"foreignKey:ChangedBy" json:"changer,omitempty"`
	Action     string         `gorm:"not null;size:20" json:"action"` // CREATE, UPDATE, DELETE
	OldData    datatypes.JSON `json:"old_data,omitempty"`
	NewData    datatypes.JSON `json:"new_data,omitempty"`
	IPAddress  string         `gorm:"size:45" json:"ip_address,omitempty"`
	CreatedAt  time.Time      `gorm:"not null" json:"created_at"`
}
