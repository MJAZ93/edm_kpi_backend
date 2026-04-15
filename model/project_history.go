package model

import "gorm.io/gorm"

type ProjectHistory struct {
	gorm.Model
	ProjectID       uint     `gorm:"not null" json:"project_id"`
	Project         *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Value           float64  `gorm:"type:decimal(15,2);not null" json:"value"`
	PeriodReference string   `gorm:"size:20" json:"period_reference,omitempty"`
	Notes           string   `gorm:"type:text" json:"notes,omitempty"`
	CreatedBy       uint     `gorm:"not null" json:"created_by"`
	Creator         *User    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
