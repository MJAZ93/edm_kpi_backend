package dao

import (
	"encoding/json"
	"kpi-backend/model"
	"time"

	"gorm.io/datatypes"
)

type AuditDao struct{}

func (d *AuditDao) Write(entityType string, entityID, changedBy uint, action string, oldData, newData interface{}, ipAddress string) error {
	var oldJSON, newJSON datatypes.JSON

	if oldData != nil {
		b, _ := json.Marshal(oldData)
		oldJSON = b
	}
	if newData != nil {
		b, _ := json.Marshal(newData)
		newJSON = b
	}

	log := model.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		ChangedBy:  changedBy,
		Action:     action,
		OldData:    oldJSON,
		NewData:    newJSON,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
	}
	return Database.Create(&log).Error
}

func (d *AuditDao) List(page, limit int, filters map[string]interface{}) ([]model.AuditLog, int64, error) {
	var list []model.AuditLog
	var total int64

	q := Database.Model(&model.AuditLog{})
	for k, v := range filters {
		q = q.Where(k+" = ?", v)
	}
	q.Count(&total)

	q2 := Database.Preload("Changer")
	for k, v := range filters {
		q2 = q2.Where(k+" = ?", v)
	}
	if limit > 0 && page >= 0 {
		q2 = q2.Offset(page * limit).Limit(limit)
	}
	err := q2.Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (d *AuditDao) ListByEntity(entityType string, entityID uint, page, limit int) ([]model.AuditLog, int64, error) {
	filters := map[string]interface{}{
		"entity_type": entityType,
		"entity_id":   entityID,
	}
	return d.List(page, limit, filters)
}
