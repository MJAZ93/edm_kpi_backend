package dao

import "kpi-backend/model"

type ProjectDao struct{}

func (d *ProjectDao) Create(p *model.Project) error {
	return Database.Create(p).Error
}

func (d *ProjectDao) GetByID(id uint) (model.Project, error) {
	var p model.Project
	err := Database.Preload("Creator").Preload("Parent").Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *ProjectDao) List(page, limit int, filters map[string]interface{}) ([]model.Project, int64, error) {
	var list []model.Project
	var total int64

	q := Database.Model(&model.Project{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	q.Count(&total)

	q2 := Database.Preload("Creator")
	for k, v := range filters {
		q2 = q2.Where(k+" = ?", v)
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *ProjectDao) GetChildren(parentID uint) ([]model.Project, error) {
	var list []model.Project
	err := Database.Preload("Creator").Where("parent_id = ?", parentID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (d *ProjectDao) GetTree(id uint) (model.Project, error) {
	var p model.Project
	err := Database.
		Preload("Creator").
		Preload("Tasks").
		Preload("Tasks.Milestones").
		Preload("Tasks.Scopes").
		Preload("Children").
		Preload("Children.Tasks").
		Preload("Children.Tasks.Milestones").
		Preload("Children.Children").
		Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *ProjectDao) Update(p *model.Project) error {
	return Database.Save(p).Error
}

func (d *ProjectDao) SoftDelete(id uint) error {
	return Database.Delete(&model.Project{}, id).Error
}

func (d *ProjectDao) ListByCreatorType(creatorType string) ([]model.Project, error) {
	var list []model.Project
	err := Database.Preload("Creator").Where("creator_type = ?", creatorType).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (d *ProjectDao) ListByCreatorOrg(creatorType string, orgID uint) ([]model.Project, error) {
	var list []model.Project
	err := Database.Preload("Creator").
		Where("creator_type = ? AND creator_org_id = ?", creatorType, orgID).
		Order("created_at DESC").Find(&list).Error
	return list, err
}
