package dao

import (
	"kpi-backend/model"
	"time"
)

type BlockerDao struct{}

func (d *BlockerDao) Create(b *model.Blocker) error {
	autoApprove := time.Now().AddDate(0, 0, b.SLADays)
	b.AutoApproveAt = &autoApprove
	return Database.Create(b).Error
}

func (d *BlockerDao) GetByID(id uint) (model.Blocker, error) {
	var b model.Blocker
	err := Database.Preload("Reporter").Preload("Approver").Where("id = ?", id).First(&b).Error
	return b, err
}

func (d *BlockerDao) List(page, limit int, filters map[string]interface{}) ([]model.Blocker, int64, error) {
	var list []model.Blocker
	var total int64

	q := Database.Model(&model.Blocker{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	q.Count(&total)

	q2 := Database.Preload("Reporter").Preload("Approver")
	for k, v := range filters {
		q2 = q2.Where(k+" = ?", v)
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *BlockerDao) ListScoped(page, limit int, filters map[string]interface{}, scope *UserScope) ([]model.Blocker, int64, error) {
	var list []model.Blocker
	var total int64

	q := Database.Model(&model.Blocker{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	if !scope.IsGlobal {
		// Blockers are on tasks or milestones; filter by visible task IDs
		q = q.Where(
			"(entity_type = 'TASK' AND entity_id IN (?)) OR (entity_type = 'MILESTONE' AND entity_id IN (SELECT id FROM milestones WHERE task_id IN (?)))",
			scope.TaskIDsSubquery(), scope.TaskIDsSubquery(),
		)
	}
	q.Count(&total)

	q2 := Database.Preload("Reporter").Preload("Approver")
	for k, v := range filters {
		q2 = q2.Where(k+" = ?", v)
	}
	if !scope.IsGlobal {
		q2 = q2.Where(
			"(entity_type = 'TASK' AND entity_id IN (?)) OR (entity_type = 'MILESTONE' AND entity_id IN (SELECT id FROM milestones WHERE task_id IN (?)))",
			scope.TaskIDsSubquery(), scope.TaskIDsSubquery(),
		)
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *BlockerDao) Approve(id, approvedBy uint) error {
	now := time.Now()
	return Database.Model(&model.Blocker{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "APPROVED",
		"approved_by": approvedBy,
		"resolved_at": now,
	}).Error
}

func (d *BlockerDao) Reject(id, approvedBy uint, reason string) error {
	now := time.Now()
	return Database.Model(&model.Blocker{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           "REJECTED",
		"approved_by":      approvedBy,
		"rejection_reason": reason,
		"resolved_at":      now,
	}).Error
}

func (d *BlockerDao) ListPendingExpired() ([]model.Blocker, error) {
	var list []model.Blocker
	err := Database.Where("status = 'PENDING' AND auto_approve_at <= ?", time.Now()).Find(&list).Error
	return list, err
}

func (d *BlockerDao) AutoApprove(id uint) error {
	now := time.Now()
	return Database.Model(&model.Blocker{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "AUTO_APPROVED",
		"resolved_at": now,
	}).Error
}

func (d *BlockerDao) IsBlocked(entityType string, entityID uint) bool {
	var count int64
	Database.Model(&model.Blocker{}).
		Where("entity_type = ? AND entity_id = ? AND status IN ('APPROVED', 'AUTO_APPROVED')", entityType, entityID).
		Count(&count)
	return count > 0
}
