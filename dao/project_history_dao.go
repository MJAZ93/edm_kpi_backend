package dao

import "kpi-backend/model"

type ProjectHistoryDao struct{}

func (d *ProjectHistoryDao) Create(h *model.ProjectHistory) error {
	return Database.Create(h).Error
}

func (d *ProjectHistoryDao) ListByProject(projectID uint) ([]model.ProjectHistory, error) {
	var list []model.ProjectHistory
	err := Database.Preload("Creator").
		Where("project_id = ?", projectID).
		Order("period_reference ASC").
		Find(&list).Error
	return list, err
}

func (d *ProjectHistoryDao) GetByID(id uint) (model.ProjectHistory, error) {
	var h model.ProjectHistory
	err := Database.Preload("Creator").Where("id = ?", id).First(&h).Error
	return h, err
}

func (d *ProjectHistoryDao) Update(h *model.ProjectHistory) error {
	return Database.Save(h).Error
}

func (d *ProjectHistoryDao) Delete(id uint) error {
	return Database.Delete(&model.ProjectHistory{}, id).Error
}
