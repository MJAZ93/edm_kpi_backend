package model

import "gorm.io/gorm"

type Feedback struct {
	gorm.Model
	SenderID   uint       `gorm:"not null" json:"sender_id"`
	Sender     *User      `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	ReceiverID uint       `gorm:"not null" json:"receiver_id"`
	Receiver   *User      `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
	ParentID   *uint      `json:"parent_id,omitempty"`
	Parent     *Feedback  `gorm:"foreignKey:ParentID" json:"-"`
	TargetType string     `gorm:"size:30" json:"target_type,omitempty"` // DEPARTAMENTO, DIRECAO, PELOURO, USER
	TargetID   *uint      `json:"target_id,omitempty"`
	TargetName string     `gorm:"-" json:"target_name,omitempty"` // Computed, not stored
	Message    string     `gorm:"type:text;not null" json:"message"`
	Category   string     `gorm:"size:30;default:GENERAL" json:"category"` // GENERAL, PERFORMANCE, IMPROVEMENT, RECOGNITION
	IsRead     bool       `gorm:"default:false" json:"is_read"`
	Replies    []Feedback `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}
