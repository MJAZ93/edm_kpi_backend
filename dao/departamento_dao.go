package dao

import "kpi-backend/model"

type DepartamentoDao struct{}

func (d *DepartamentoDao) Create(dept *model.Departamento) error {
	return Database.Create(dept).Error
}

func (d *DepartamentoDao) GetByID(id uint) (model.Departamento, error) {
	var dept model.Departamento
	err := Database.Preload("Responsible").Preload("Direcao").Preload("Users").Where("id = ?", id).First(&dept).Error
	return dept, err
}

func (d *DepartamentoDao) List(page, limit int) ([]model.Departamento, int64, error) {
	var list []model.Departamento
	var total int64

	Database.Model(&model.Departamento{}).Count(&total)
	q := Database.Preload("Responsible").Preload("Direcao")
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *DepartamentoDao) ListByDirecao(direcaoID uint) ([]model.Departamento, error) {
	var list []model.Departamento
	err := Database.Preload("Responsible").Where("direcao_id = ?", direcaoID).Order("name ASC").Find(&list).Error
	return list, err
}

func (d *DepartamentoDao) Update(dept *model.Departamento) error {
	return Database.Save(dept).Error
}

func (d *DepartamentoDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Departamento{}, id).Error
}

func (d *DepartamentoDao) AddUser(departamentoID, userID uint) error {
	du := model.DepartamentoUser{UserID: userID, DepartamentoID: departamentoID}
	return Database.Create(&du).Error
}

func (d *DepartamentoDao) RemoveUser(departamentoID, userID uint) error {
	return Database.Where("user_id = ? AND departamento_id = ?", userID, departamentoID).Delete(&model.DepartamentoUser{}).Error
}

func (d *DepartamentoDao) GetUsers(departamentoID uint) ([]model.User, error) {
	var users []model.User
	err := Database.Table("users").
		Joins("JOIN departamento_users ON departamento_users.user_id = users.id").
		Where("departamento_users.departamento_id = ?", departamentoID).
		Find(&users).Error
	return users, err
}

func (d *DepartamentoDao) GetByResponsible(userID uint) (model.Departamento, error) {
	var dept model.Departamento
	err := Database.Where("responsible_id = ?", userID).First(&dept).Error
	return dept, err
}
