package model

import "gorm.io/gorm"

type Pelouro struct {
	gorm.Model
	Name          string    `gorm:"not null;size:255" json:"name"`
	Description   string    `gorm:"type:text" json:"description,omitempty"`
	ResponsibleID *uint     `json:"responsible_id,omitempty"`
	Responsible   *User     `gorm:"foreignKey:ResponsibleID" json:"responsible,omitempty"`
	CreatedBy     uint      `json:"created_by"`
	Creator       *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Direcoes      []Direcao `gorm:"foreignKey:PelouroID" json:"direcoes,omitempty"`
}
