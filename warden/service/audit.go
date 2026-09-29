package service

import (
	"github.com/gaucho-racing/ulid-go"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

// RecordAudit persists an audit row. Audit writes never fail a request —
// losing the log is bad, but refusing a link because the log write failed
// is worse.
func RecordAudit(entry model.AuditLog) {
	if entry.ID == "" {
		entry.ID = ulid.Make().Prefixed("wlog")
	}
	if err := database.DB.Create(&entry).Error; err != nil {
		logger.SugarLogger.Errorf("failed to write audit log %s: %v", entry.Action, err)
	}
}

func ListAuditLogs(limit int) ([]model.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	logs := []model.AuditLog{}
	if err := database.DB.Order("created_at desc").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
