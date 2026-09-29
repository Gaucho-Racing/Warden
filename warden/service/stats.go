package service

import (
	"errors"
	"time"

	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// snapshotRetention bounds the history table. Long enough for any window the
// portal shows, short enough that the table stays trivial.
const snapshotRetention = 120 * 24 * time.Hour

var ErrStatsNotFound = errors.New("no stats reported for this player")

// RecordPlayerStats stores a report from the plugin and, if this is the
// first report of the UTC day, snapshots the counters.
//
// Snapshot-on-ingest rather than on a schedule: a snapshot only needs to
// exist for days somebody played, and there is no cron to fail quietly.
func RecordPlayerStats(stats model.PlayerStats) error {
	stats.ReportedAt = time.Now()
	if stats.LastSeen.IsZero() {
		stats.LastSeen = stats.ReportedAt
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// FirstSeen is set once and never moved forward by a later report.
		var existing model.PlayerStats
		err := tx.Where("uuid = ?", stats.UUID).First(&existing).Error
		if err == nil && !existing.FirstSeen.IsZero() {
			stats.FirstSeen = existing.FirstSeen
		} else if stats.FirstSeen.IsZero() {
			stats.FirstSeen = stats.ReportedAt
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uuid"}},
			UpdateAll: true,
		}).Create(&stats).Error; err != nil {
			return err
		}

		day := stats.ReportedAt.UTC().Truncate(24 * time.Hour)
		snapshot := model.PlayerStatsSnapshot{
			UUID:               stats.UUID,
			Day:                day,
			PlayerStatCounters: stats.PlayerStatCounters,
		}
		// DoNothing, not UpdateAll: the snapshot should record where the
		// player stood at the start of the day, so the first report wins.
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uuid"}, {Name: "day"}},
			DoNothing: true,
		}).Create(&snapshot).Error; err != nil {
			return err
		}

		return tx.Where("day < ?", time.Now().Add(-snapshotRetention)).
			Delete(&model.PlayerStatsSnapshot{}).Error
	})
}

// PlayerStatsForUUID returns the stat line for a linked account.
//
// A linked player who has never been reported on comes back as a zeroed row
// marked StatSourceNone, so the portal can say the stats have not arrived
// yet rather than presenting zeroes as fact.
func PlayerStatsForUUID(uuid string) (model.PlayerStats, error) {
	account, err := GetAccountByUUID(uuid)
	if err != nil {
		return model.PlayerStats{}, err
	}

	var stats model.PlayerStats
	err = database.DB.Where("uuid = ?", uuid).First(&stats).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PlayerStats{
			UUID:     account.UUID,
			Username: account.Username,
			Source:   model.StatSourceNone,
		}, nil
	}
	if err != nil {
		return model.PlayerStats{}, err
	}

	stats.Source = model.StatSourcePlugin
	stats.Last7Days = windowSince(uuid, stats, 7*24*time.Hour)
	return stats, nil
}

// windowSince subtracts the newest snapshot at or before the cutoff.
//
// Returns nil rather than a zeroed window when there is no snapshot old
// enough: a player three days into their first week has no 7-day history,
// and reporting "+0" there would be a claim rather than an absence.
func windowSince(uuid string, current model.PlayerStats, age time.Duration) *model.StatWindow {
	cutoff := time.Now().UTC().Add(-age).Truncate(24 * time.Hour)

	var past model.PlayerStatsSnapshot
	err := database.DB.Where("uuid = ? AND day <= ?", uuid, cutoff).
		Order("day desc").
		First(&past).Error
	if err != nil {
		return nil
	}

	return &model.StatWindow{
		PlaytimeMinutes: current.PlaytimeMinutes - past.PlaytimeMinutes,
		Deaths:          current.Deaths - past.Deaths,
		MobKills:        current.MobKills - past.MobKills,
		BlocksMined:     current.BlocksMined - past.BlocksMined,
		DistanceMeters:  current.DistanceMeters - past.DistanceMeters,
	}
}
