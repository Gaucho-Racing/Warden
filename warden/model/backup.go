package model

import "time"

// Backup job statuses. A job is "active" in any of the first three; exactly
// one active job may exist at a time, because the archive is written to the
// game server's own disk and two at once would both compete for space and
// capture each other's temporary files.
const (
	BackupStatusPending   = "pending"
	BackupStatusArchiving = "archiving"
	BackupStatusUploading = "uploading"
	BackupStatusSucceeded = "succeeded"
	BackupStatusFailed    = "failed"
	BackupStatusTimedOut  = "timed_out"
)

const (
	BackupTriggerSchedule = "schedule"
	BackupTriggerManual   = "manual"
)

// BackupJob is one attempt at archiving the game server and uploading it to
// Depot.
//
// Warden orchestrates but never touches the bytes: the world lives on a
// ReadWriteOnce volume mounted only into the game server pod, so the plugin
// is the only party that can read it. Warden mints a presigned upload URL
// from Depot, hands it to the plugin, and records what the plugin reports
// back. The archive therefore goes straight from the game server to object
// storage without passing through either Warden or Depot.
type BackupJob struct {
	ID      string `json:"id" gorm:"primaryKey"`
	Status  string `json:"status" gorm:"index"`
	Trigger string `json:"trigger"`
	// RequestedByEntityID is set for manual runs and empty for scheduled ones.
	RequestedByEntityID string `json:"requested_by_entity_id" gorm:"index"`

	// ScheduledFor is the cron occurrence this job covers, and is unique so a
	// second Warden replica cannot double-fire the same tick. Null for manual
	// runs, which are deliberately unconstrained — Postgres does not collide
	// nulls in a unique index.
	ScheduledFor *time.Time `json:"scheduled_for,omitempty" gorm:"uniqueIndex"`

	// Depot coordinates. The file row exists from the moment the upload is
	// initiated, so a failed job still points at the pending file it orphaned.
	DepotFileID string `json:"depot_file_id"`
	DepotBucket string `json:"depot_bucket"`
	FileName    string `json:"file_name"`

	SizeBytes int64 `json:"size_bytes"`
	// Archive and upload durations as the plugin measured them, which is the
	// only place the split is visible.
	ArchiveMillis int64  `json:"archive_millis"`
	UploadMillis  int64  `json:"upload_millis"`
	Error         string `json:"error,omitempty"`

	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (BackupJob) TableName() string {
	return "warden_backup_job"
}

func (j BackupJob) Active() bool {
	switch j.Status {
	case BackupStatusPending, BackupStatusArchiving, BackupStatusUploading:
		return true
	}
	return false
}

// BackupScheduleID is the primary key of the single schedule row. Warden
// backs up one game server, so a table with a fixed key is a simpler home
// for an editable setting than a key/value settings table would be.
const BackupScheduleID = "default"

// BackupSchedule is when automatic backups run. Stored rather than
// configured through the environment so MinecraftAdmins can change it from
// the portal without a redeploy.
type BackupSchedule struct {
	ID string `json:"id" gorm:"primaryKey"`
	// Cron is a standard five-field expression, or one of the @daily-style
	// descriptors. Interpreted in Timezone, never in the pod's clock zone.
	Cron     string `json:"cron"`
	Timezone string `json:"timezone"`
	Enabled  bool   `json:"enabled"`

	UpdatedByEntityID string    `json:"updated_by_entity_id"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (BackupSchedule) TableName() string {
	return "warden_backup_schedule"
}
