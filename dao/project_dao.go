package dao

import "kpi-backend/model"

type ProjectDao struct{}

func (d *ProjectDao) Create(p *model.Project) error {
	return Database.Create(p).Error
}

func (d *ProjectDao) GetByID(id uint) (model.Project, error) {
	var p model.Project
	err := Database.Preload("Creator").Preload("Parent").Preload("Direcoes").Where("id = ?", id).First(&p).Error
	return p, err
}

func (d *ProjectDao) SetDirecoes(projectID uint, direcaoIDs []uint) error {
	direcoes := make([]model.Direcao, len(direcaoIDs))
	for i, id := range direcaoIDs {
		direcoes[i].ID = id
	}
	project := model.Project{}
	project.ID = projectID
	return Database.Model(&project).Association("Direcoes").Replace(direcoes)
}

func (d *ProjectDao) List(page, limit int, filters map[string]interface{}) ([]model.Project, int64, error) {
	var list []model.Project
	var total int64

	q := Database.Model(&model.Project{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	q.Count(&total)

	q2 := Database.Preload("Creator").Preload("Tasks").Preload("Tasks.Milestones").Preload("Tasks.Scopes")
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
		Preload("Direcoes").
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

func (d *ProjectDao) ListScoped(page, limit int, filters map[string]interface{}, scope *UserScope) ([]model.Project, int64, error) {
	var list []model.Project
	var total int64

	q := Database.Model(&model.Project{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	q = scope.ApplyToProjects(q)
	q.Count(&total)

	q2 := Database.Preload("Creator").Preload("Tasks").Preload("Tasks.Milestones").Preload("Tasks.Scopes")
	for k, v := range filters {
		q2 = q2.Where(k+" = ?", v)
	}
	q2 = scope.ApplyToProjects(q2)
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
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

// ListByDirecao returns projects linked to a given direcao, via:
//  1. project_direcoes join table
//  2. tasks directly owned by the direcao
//  3. tasks owned by a departamento that belongs to this direcao
func (d *ProjectDao) ListByDirecao(direcaoID uint) ([]model.Project, error) {
	type row struct{ ProjectID uint }
	var rows1, rows2, rows3 []row

	Database.Raw(`SELECT project_id FROM project_direcoes WHERE direcao_id = ?`, direcaoID).Scan(&rows1)
	Database.Raw(`SELECT DISTINCT project_id FROM tasks WHERE owner_type = 'DIRECAO' AND owner_id = ? AND deleted_at IS NULL`, direcaoID).Scan(&rows2)
	Database.Raw(`SELECT DISTINCT t.project_id FROM tasks t JOIN departamentos d ON d.id = t.owner_id WHERE t.owner_type = 'DEPARTAMENTO' AND d.direcao_id = ? AND t.deleted_at IS NULL AND d.deleted_at IS NULL`, direcaoID).Scan(&rows3)

	seen := map[uint]bool{}
	var ids []uint
	for _, r := range append(append(rows1, rows2...), rows3...) {
		if !seen[r.ProjectID] {
			seen[r.ProjectID] = true
			ids = append(ids, r.ProjectID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	var list []model.Project
	err := Database.Preload("Creator").Preload("Direcoes").
		Preload("Tasks", "deleted_at IS NULL").Preload("Tasks.Milestones", "deleted_at IS NULL").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Order("created_at DESC").Find(&list).Error
	return list, err
}
