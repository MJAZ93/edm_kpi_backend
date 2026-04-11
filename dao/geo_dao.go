package dao

import "kpi-backend/model"

type GeoDao struct{}

// --- Regiões ---

func (d *GeoDao) CreateRegiao(r *model.Regiao) error {
	return Database.Create(r).Error
}

func (d *GeoDao) GetRegiaoByID(id uint) (model.Regiao, error) {
	var r model.Regiao
	err := Database.Preload("Responsible").Where("id = ?", id).First(&r).Error
	return r, err
}

func (d *GeoDao) ListRegioes(page, limit int) ([]model.Regiao, int64, error) {
	var list []model.Regiao
	var total int64

	Database.Model(&model.Regiao{}).Count(&total)
	q := Database.Preload("Responsible")
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *GeoDao) UpdateRegiao(r *model.Regiao) error {
	return Database.Save(r).Error
}

func (d *GeoDao) DeleteRegiao(id uint) error {
	return Database.Delete(&model.Regiao{}, id).Error
}

func (d *GeoDao) GetRegiaoByResponsible(userID uint) (model.Regiao, error) {
	var r model.Regiao
	err := Database.Preload("Responsible").Where("responsible_id = ?", userID).First(&r).Error
	return r, err
}

func (d *GeoDao) GetAllRegioes() ([]model.Regiao, error) {
	var list []model.Regiao
	err := Database.Preload("Responsible").Order("name ASC").Find(&list).Error
	return list, err
}

// GetAllRegioesLight returns only id, name, code — no relations, no polygon.
func (d *GeoDao) GetAllRegioesLight() ([]model.Regiao, error) {
	var list []model.Regiao
	err := Database.Select("id, name, code").Order("name ASC").Find(&list).Error
	return list, err
}

// --- ASCs ---

func (d *GeoDao) CreateASC(a *model.ASC) error {
	return Database.Create(a).Error
}

func (d *GeoDao) GetASCByID(id uint) (model.ASC, error) {
	var a model.ASC
	err := Database.Preload("Responsible").Preload("Director").Preload("Regiao").Where("id = ?", id).First(&a).Error
	return a, err
}

func (d *GeoDao) ListASCs(page, limit int) ([]model.ASC, int64, error) {
	var list []model.ASC
	var total int64

	Database.Model(&model.ASC{}).Count(&total)
	q := Database.Preload("Responsible").Preload("Director").Preload("Regiao")
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *GeoDao) ListASCsByRegiao(regiaoID uint) ([]model.ASC, error) {
	var list []model.ASC
	err := Database.Preload("Responsible").Preload("Director").Where("regiao_id = ?", regiaoID).Order("name ASC").Find(&list).Error
	return list, err
}

func (d *GeoDao) UpdateASC(a *model.ASC) error {
	return Database.Save(a).Error
}

func (d *GeoDao) DeleteASC(id uint) error {
	return Database.Delete(&model.ASC{}, id).Error
}

func (d *GeoDao) GetASCByDirector(userID uint) (model.ASC, error) {
	var a model.ASC
	err := Database.Where("director_id = ?", userID).First(&a).Error
	return a, err
}

func (d *GeoDao) GetAllASCs() ([]model.ASC, error) {
	var list []model.ASC
	err := Database.Preload("Responsible").Preload("Director").Preload("Regiao").Order("name ASC").Find(&list).Error
	return list, err
}

// GetAllASCsLight returns only id, name, code, regiao_id — no relations, no polygon.
func (d *GeoDao) GetAllASCsLight() ([]model.ASC, error) {
	var list []model.ASC
	err := Database.Select("id, name, code, regiao_id").Order("name ASC").Find(&list).Error
	return list, err
}
