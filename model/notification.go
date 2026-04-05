package model

import "time"

type Notification struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null" json:"user_id"`
	User       *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title      string    `gorm:"not null;size:255" json:"title"`
	Message    string    `gorm:"type:text;not null" json:"message"`
	Type       string    `gorm:"not null;size:40" json:"type"` // TASK_UPDATE, MILESTONE_UPDATE, BLOCKER_CREATED, BLOCKER_RESOLVED, FORECAST_RISK, DELAY_ALERT, MILESTONE_OVERDUE, GOAL_AT_RISK
	EntityType string    `gorm:"size:30" json:"entity_type,omitempty"`
	EntityID   *uint     `json:"entity_id,omitempty"`
	IsRead     bool      `gorm:"default:false" json:"is_read"`
	EmailSent  bool      `gorm:"default:false" json:"email_sent"`
	EmailSentAt *time.Time `json:"email_sent_at,omitempty"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
}
