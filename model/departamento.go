package model

import "gorm.io/gorm"

type Departamento struct {
	gorm.Model
	Name          string `gorm:"not null;size:255" json:"name"`
	Description   string `gorm:"type:text" json:"description,omitempty"`
	DirecaoID     uint   `gorm:"not null" json:"direcao_id"`
	Direcao       *Direcao `gorm:"foreignKey:DirecaoID" json:"direcao,omitempty"`
	ResponsibleID *uint  `json:"responsible_id,omitempty"`
	Responsible   *User  `gorm:"foreignKey:ResponsibleID" json:"responsible,omitempty"`
	CreatedBy     uint   `json:"created_by"`
	Creator       *User  `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Users         []User `gorm:"many2many:departamento_users;" json:"users,omitempty"`
}

type DepartamentoUser struct {
	UserID         uint `gorm:"primaryKey" json:"user_id"`
	DepartamentoID uint `gorm:"primaryKey" json:"departamento_id"`
}

func (DepartamentoUser) TableName() string {
	return "departamento_users"
}
