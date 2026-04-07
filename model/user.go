package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name                 string     `gorm:"not null;size:128" json:"name"`
	Email                string     `gorm:"not null;size:255;uniqueIndex" json:"email"`
	Password             string     `gorm:"not null;size:255" json:"-"`
	Role                 string     `gorm:"not null;size:20" json:"role"` // ADMIN, CA, PELOURO, DIRECAO, DEPARTAMENTO
	Active               bool       `gorm:"default:true" json:"active"`
	PasswordResetToken   string     `gorm:"size:255" json:"-"`
	PasswordResetExpires *time.Time `json:"-"`
	LastLogin            *time.Time `json:"last_login,omitempty"`
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.Password == "" {
		return nil
	}
	// Only hash if password changed (not already a bcrypt hash)
	if len(u.Password) < 60 || u.Password[:4] != "$2a$" {
		b, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(b)
	}
	return nil
}

type UserResponse struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	Active        bool       `json:"active"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	// DirectorScope is only set for DIRECAO role users:
	// "DIRECTION" — responsible for a Direcção (full write access)
	// "REGION"    — responsible for a Região only (read-only)
	// ""          — not configured as responsible for anything
	DirectorScope string     `json:"director_scope,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		Active:    u.Active,
		LastLogin: u.LastLogin,
		CreatedAt: u.CreatedAt,
	}
}
