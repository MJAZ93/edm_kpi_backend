package model

import "gorm.io/gorm"

type Regiao struct {
	gorm.Model
	Name          string `gorm:"not null;size:255" json:"name"`
	Code          string `gorm:"size:50" json:"code,omitempty"`
	Polygon       string `gorm:"type:text" json:"polygon,omitempty"` // GeoJSON stored as text; PostGIS ops via raw SQL
	ResponsibleID *uint  `json:"responsible_id,omitempty"`
	Responsible   *User  `gorm:"foreignKey:ResponsibleID" json:"responsible,omitempty"`
	ASCs          []ASC  `gorm:"foreignKey:RegiaoID" json:"ascs,omitempty"`
}

type ASC struct {
	gorm.Model
	Name          string  `gorm:"not null;size:255" json:"name"`
	Code          string  `gorm:"size:50" json:"code,omitempty"`
	RegiaoID      *uint   `json:"regiao_id,omitempty"`
	Regiao        *Regiao `gorm:"foreignKey:RegiaoID" json:"regiao,omitempty"`
	Polygon       string  `gorm:"type:text" json:"polygon,omitempty"`
	ResponsibleID *uint   `json:"responsible_id,omitempty"`
	Responsible   *User   `gorm:"foreignKey:ResponsibleID" json:"responsible,omitempty"`
	DirectorID    *uint   `json:"director_id,omitempty"`
	Director      *User   `gorm:"foreignKey:DirectorID" json:"director,omitempty"`
}
