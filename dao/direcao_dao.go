package dao

import "kpi-backend/model"

type DirecaoDao struct{}

func (d *DirecaoDao) Create(dir *model.Direcao) error {
	return Database.Create(dir).Error
}

func (d *DirecaoDao) GetByID(id uint) (model.Direcao, error) {
	var dir model.Direcao
	err := Database.Preload("Responsible").Preload("Pelouro").Where("id = ?", id).First(&dir).Error
	return dir, err
}

func (d *DirecaoDao) List(page, limit int) ([]model.Direcao, int64, error) {
	var list []model.Direcao
	var total int64

	Database.Model(&model.Direcao{}).Count(&total)
	q := Database.Preload("Responsible").Preload("Pelouro")
	if limit > 0 && page >= 0 {
		q = q.Offset(page * limit).Limit(limit)
	}
	err := q.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *DirecaoDao) ListScoped(page, limit int, scope *UserScope) ([]model.Direcao, int64, error) {
	var list []model.Direcao
	var total int64

	q := Database.Model(&model.Direcao{})
	if !scope.IsGlobal && len(scope.DirecaoIDs) > 0 {
		q = q.Where("id IN ?", scope.DirecaoIDs)
	} else if !scope.IsGlobal {
		q = q.Where("id IN ?", []uint{0})
	}
	q.Count(&total)

	q2 := Database.Preload("Responsible").Preload("Pelouro")
	if !scope.IsGlobal && len(scope.DirecaoIDs) > 0 {
		q2 = q2.Where("id IN ?", scope.DirecaoIDs)
	} else if !scope.IsGlobal {
		q2 = q2.Where("id IN ?", []uint{0})
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("name ASC").Find(&list).Error
	return list, total, err
}

func (d *DirecaoDao) ListByPelouro(pelouroID uint) ([]model.Direcao, error) {
	var list []model.Direcao
	err := Database.Preload("Responsible").Where("pelouro_id = ?", pelouroID).Order("name ASC").Find(&list).Error
	return list, err
}

func (d *DirecaoDao) Update(dir *model.Direcao) error {
	return Database.Save(dir).Error
}

func (d *DirecaoDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Direcao{}, id).Error
}

func (d *DirecaoDao) GetByResponsible(userID uint) (model.Direcao, error) {
	var dir model.Direcao
	err := Database.Where("responsible_id = ?", userID).First(&dir).Error
	return dir, err
}
