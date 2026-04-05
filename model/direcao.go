package model

import "gorm.io/gorm"

type Direcao struct {
	gorm.Model
	Name           string         `gorm:"not null;size:255" json:"name"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	PelouroID      uint           `gorm:"not null" json:"pelouro_id"`
	Pelouro        *Pelouro       `gorm:"foreignKey:PelouroID" json:"pelouro,omitempty"`
	ResponsibleID  *uint          `json:"responsible_id,omitempty"`
	Responsible    *User          `gorm:"foreignKey:ResponsibleID" json:"responsible,omitempty"`
	CreatedBy      uint           `json:"created_by"`
	Creator        *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Departamentos  []Departamento `gorm:"foreignKey:DirecaoID" json:"departamentos,omitempty"`
}
