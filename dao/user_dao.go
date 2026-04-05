package dao

import (
	"kpi-backend/model"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserDao struct{ Limit int }

func (d *UserDao) Create(u *model.User) error {
	return Database.Create(u).Error
}

func (d *UserDao) GetByID(id uint) (model.User, error) {
	var u model.User
	err := Database.Where("id = ?", id).First(&u).Error
	return u, err
}

func (d *UserDao) GetByEmail(email string) (model.User, error) {
	var u model.User
	err := Database.Where("email = ?", email).First(&u).Error
	return u, err
}

func (d *UserDao) GetByEmailAndPassword(email, password string) (model.User, error) {
	var u model.User
	if err := Database.Where("email = ? AND active = true", email).First(&u).Error; err != nil {
		return model.User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (d *UserDao) List(page, limit int) ([]model.User, int64, error) {
	var list []model.User
	var total int64

	Database.Model(&model.User{}).Count(&total)

	q := Database
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *UserDao) Update(u *model.User) error {
	return Database.Save(u).Error
}

func (d *UserDao) SoftDelete(id uint) error {
	return Database.Delete(&model.User{}, id).Error
}

func (d *UserDao) SetPasswordResetToken(userID uint) (string, error) {
	token := uuid.New().String()
	expires := time.Now().Add(1 * time.Hour)
	err := Database.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_reset_token":   token,
		"password_reset_expires": expires,
	}).Error
	return token, err
}

func (d *UserDao) GetByResetToken(token string) (model.User, error) {
	var u model.User
	err := Database.Where("password_reset_token = ? AND password_reset_expires > ?", token, time.Now()).First(&u).Error
	return u, err
}

func (d *UserDao) ClearResetToken(userID uint) error {
	return Database.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_reset_token":   nil,
		"password_reset_expires": nil,
	}).Error
}

func (d *UserDao) UpdateLastLogin(userID uint) {
	now := time.Now()
	Database.Model(&model.User{}).Where("id = ?", userID).Update("last_login", now)
}

func (d *UserDao) GetByIDs(ids []uint) ([]model.User, error) {
	var users []model.User
	err := Database.Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (d *UserDao) GetByRole(role string) ([]model.User, error) {
	var users []model.User
	err := Database.Where("role = ? AND active = true", role).Find(&users).Error
	return users, err
}
