package dao

import "kpi-backend/model"

type PelouroDao struct{}

func (d *PelouroDao) Create(p *model.Pelouro) error {
	return Database.Create(p).Error
}

func (d *PelouroDao) GetByID(id uint) (model.Pelouro, error) {
	var p model.Pelouro
	err := Database.Preload("Responsible").Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *PelouroDao) List(page, limit int) ([]model.Pelouro, int64, error) {
	var list []model.Pelouro
	var total int64

	Database.Model(&model.Pelouro{}).Count(&total)
	q := Database.Preload("Responsible")
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *PelouroDao) Update(p *model.Pelouro) error {
	return Database.Save(p).Error
}

func (d *PelouroDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Pelouro{}, id).Error
}

func (d *PelouroDao) GetAll() ([]model.Pelouro, error) {
	var list []model.Pelouro
	err := Database.Preload("Responsible").Order("name ASC").Find(&list).Error
	return list, err
}
