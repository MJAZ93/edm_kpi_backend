package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	gorm.Model
	Title        string     `gorm:"not null;size:500" json:"title"`
	Description  string     `gorm:"type:text" json:"description,omitempty"`
	CreatorType  string     `gorm:"not null;size:20" json:"creator_type"` // CA, PELOURO, DIRECAO, DEPARTAMENTO
	CreatorOrgID *uint      `json:"creator_org_id,omitempty"`             // pelouro_id / direcao_id / departamento_id; nil for CA
	ParentID     *uint      `json:"parent_id,omitempty"`
	Parent       *Project   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children     []Project  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Weight       float64    `gorm:"type:decimal(5,2);default:100.0" json:"weight"`
	Status       string     `gorm:"size:20;default:ACTIVE" json:"status"` // ACTIVE, COMPLETED, CANCELLED, ARCHIVED
	CreatedBy    uint       `gorm:"not null" json:"created_by"`
	Creator      *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Tasks        []Task     `gorm:"foreignKey:ProjectID" json:"tasks,omitempty"`
}
