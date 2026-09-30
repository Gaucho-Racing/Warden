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

// serverStatusStaleAfter: the plugin reports every minute, so missing a
// couple of reports means the server is gone even if it never said it was
// stopping (a crash, say).
const serverStatusStaleAfter = 150 * time.Second

const (
	ServerActive  = "active"
	ServerEmpty   = "empty"
	ServerOffline = "offline"
)

var (
	latestStatusMu sync.RWMutex
	latestStatus   *model.ServerStatus
	stoppingAt     time.Time
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

// MarkServerStopping records that the plugin announced a shutdown, so the
// server reads as offline immediately instead of after the stale window.
func MarkServerStopping() {
	latestStatusMu.Lock()
	defer latestStatusMu.Unlock()
	stoppingAt = time.Now()
}

// CurrentServerState is ServerActive, ServerEmpty or ServerOffline, with the
// latest sample (nil if none has arrived since Warden started).
func CurrentServerState() (string, *model.ServerStatus) {
	latestStatusMu.RLock()
	defer latestStatusMu.RUnlock()
	if latestStatus == nil {
		return ServerOffline, nil
	}
	status := *latestStatus
	switch {
	case status.RecordedAt.Before(stoppingAt) || time.Since(status.RecordedAt) > serverStatusStaleAfter:
		return ServerOffline, &status
	case status.Online == 0:
		return ServerEmpty, &status
	default:
		return ServerActive, &status
	}
}
