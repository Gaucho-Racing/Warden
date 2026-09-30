package service

import (
	"sync"
	"time"

	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"gorm.io/gorm"
)

// serverStatusRetention bounds the history table: a year of minute samples
// is roughly half a million narrow rows.
const serverStatusRetention = 365 * 24 * time.Hour

var (
	latestStatusMu sync.RWMutex
	latestStatus   *model.ServerStatus
)

// RecordServerStatus stores a sample and makes it the latest one.
func RecordServerStatus(status model.ServerStatus) error {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&status).Error; err != nil {
			return err
		}
		return tx.Where("recorded_at < ?", time.Now().Add(-serverStatusRetention)).
			Delete(&model.ServerStatus{}).Error
	})
	if err != nil {
		return err
	}
	latestStatusMu.Lock()
	defer latestStatusMu.Unlock()
	latestStatus = &status
	return nil
}

// LatestServerStatus is the most recent sample since Warden started, or nil.
func LatestServerStatus() *model.ServerStatus {
	latestStatusMu.RLock()
	defer latestStatusMu.RUnlock()
	if latestStatus == nil {
		return nil
	}
	status := *latestStatus
	return &status
}
