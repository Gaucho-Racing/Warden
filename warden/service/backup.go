package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gaucho-racing/ulid-go"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/depot"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

var (
	ErrBackupsDisabled   = errors.New("server backups are not configured")
	ErrGameServerOffline = errors.New("the game server is not connected")
	ErrBackupInProgress  = errors.New("a backup is already running")
)

// GameLink is the game server as the backup pipeline needs to see it:
// somewhere to put a notice, and a way to issue a command.
//
// The interface lives here, and the bridge registers itself into it at
// startup, because the dependency already runs the other way — the bridge
// imports this package for the server status it drives Discord from. The
// backup flow would otherwise close the loop into an import cycle.
type GameLink interface {
	Announce(text string)
	StartBackup(jobID string, uploadURL string, method string, contentType string, fileName string) error
	PluginConnected() bool
}

var (
	gameLinkMu sync.RWMutex
	gameLink   GameLink
)

func SetGameLink(link GameLink) {
	gameLinkMu.Lock()
	defer gameLinkMu.Unlock()
	gameLink = link
}

func currentGameLink() GameLink {
	gameLinkMu.RLock()
	defer gameLinkMu.RUnlock()
	return gameLink
}

// announce is best effort by design. A backup is worth running even when
// nobody can be told it is happening.
func announce(text string) {
	if link := currentGameLink(); link != nil {
		link.Announce(text)
	}
}

// BackupsEnabled is false when Warden has no Depot credential, which turns
// the whole feature off rather than letting jobs fail one at a time.
func BackupsEnabled() bool {
	return depot.Enabled()
}

// GameServerConnected reports whether a backup could be dispatched right
// now. This is the live socket, not CurrentServerState: the status samples
// behind that arrive once a minute over HTTP, so they can say "online" for
// up to two and a half minutes after the pod the command would go to has
// gone away.
func GameServerConnected() bool {
	link := currentGameLink()
	return link != nil && link.PluginConnected()
}

// ---------------------------------------------------------------- schedule

const (
	defaultBackupCron = "0 4 * * *"
	// Enough occurrences for the portal to show a schedule is doing what the
	// person writing it meant, without turning the preview into a calendar.
	maxSchedulePreview = 10
)

// ParseBackupCron validates a schedule the way the scheduler will run it.
// The zone is bound into the expression rather than applied afterwards: a
// daily 4am backup must stay at 4am across a DST shift, which only holds if
// the cron fields are evaluated in the target zone.
func ParseBackupCron(spec string, timezone string) (cron.Schedule, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, errors.New("a cron expression is required")
	}
	// The zone is ours to set. Accepting an inline one would let the two
	// disagree, and the portal would then preview the wrong times.
	if strings.HasPrefix(spec, "TZ=") || strings.HasPrefix(spec, "CRON_TZ=") {
		return nil, errors.New("set the timezone in the timezone field, not in the expression")
	}
	if timezone == "" {
		timezone = config.BackupTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("unknown timezone %q", timezone)
	}
	schedule, err := cron.ParseStandard("CRON_TZ=" + timezone + " " + spec)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	return schedule, nil
}

// NextBackupRuns is what the portal previews. Computed here rather than in
// the browser so the preview and the scheduler can never disagree about
// what an expression means.
func NextBackupRuns(spec string, timezone string, count int) ([]time.Time, error) {
	schedule, err := ParseBackupCron(spec, timezone)
	if err != nil {
		return nil, err
	}
	if count < 1 {
		count = 1
	}
	if count > maxSchedulePreview {
		count = maxSchedulePreview
	}
	runs := make([]time.Time, 0, count)
	at := time.Now()
	for range count {
		at = schedule.Next(at)
		// A descriptor like "@every 0s" cannot advance; stop rather than spin.
		if at.IsZero() {
			break
		}
		runs = append(runs, at)
	}
	return runs, nil
}

// GetBackupSchedule reads the single schedule row, creating it on first use
// so the portal always has something to edit.
func GetBackupSchedule() (model.BackupSchedule, error) {
	schedule := model.BackupSchedule{
		ID:       model.BackupScheduleID,
		Cron:     defaultBackupCron,
		Timezone: config.BackupTimezone,
		Enabled:  true,
	}
	err := database.DB.Where(model.BackupSchedule{ID: model.BackupScheduleID}).
		Attrs(schedule).
		FirstOrCreate(&schedule).Error
	if err != nil {
		return model.BackupSchedule{}, err
	}
	return schedule, nil
}

func SaveBackupSchedule(spec string, timezone string, enabled bool, actorEntityID string) (model.BackupSchedule, error) {
	spec = strings.TrimSpace(spec)
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = config.BackupTimezone
	}
	if _, err := ParseBackupCron(spec, timezone); err != nil {
		return model.BackupSchedule{}, err
	}

	schedule := model.BackupSchedule{
		ID:                model.BackupScheduleID,
		Cron:              spec,
		Timezone:          timezone,
		Enabled:           enabled,
		UpdatedByEntityID: actorEntityID,
	}
	if err := database.DB.Save(&schedule).Error; err != nil {
		return model.BackupSchedule{}, err
	}
	return schedule, nil
}

// --------------------------------------------------------------- job state

func ListBackupJobs(limit int) ([]model.BackupJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	jobs := []model.BackupJob{}
	if err := database.DB.Order("created_at desc").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetBackupJob(id string) (model.BackupJob, error) {
	var job model.BackupJob
	if err := database.DB.Where("id = ?", id).First(&job).Error; err != nil {
		return model.BackupJob{}, err
	}
	return job, nil
}

var activeBackupStatuses = []string{
	model.BackupStatusPending,
	model.BackupStatusArchiving,
	model.BackupStatusUploading,
}

// ActiveBackupJob is the one job currently in flight, or nil. Only one may
// run at a time: the archive is staged on the game server's own disk, and a
// second run would both compete for that space and capture the first run's
// half-written file.
func ActiveBackupJob() (*model.BackupJob, error) {
	return firstBackupJob(database.DB.Where("status IN ?", activeBackupStatuses))
}

func LastFinishedBackup() (*model.BackupJob, error) {
	return firstBackupJob(database.DB.Where("status NOT IN ?", activeBackupStatuses))
}

// firstBackupJob returns the newest matching job, or nil when there is none.
//
// Find into a slice rather than First into a struct: no rows is the normal
// answer here, and First raises ErrRecordNotFound for it, which gorm logs at
// error level. The backups page polls both of these every few seconds, so
// that is a steady stream of error-shaped lines for a server that simply has
// not been backed up yet.
func firstBackupJob(query *gorm.DB) (*model.BackupJob, error) {
	jobs := []model.BackupJob{}
	if err := query.Order("created_at desc").Limit(1).Find(&jobs).Error; err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	return &jobs[0], nil
}

// ------------------------------------------------------------- job running

// RequestBackup creates a job and starts it in the background. It returns
// as soon as the job is recorded, because the run itself takes minutes:
// callers poll the job rather than hold a request open.
//
// delay is the grace period between warning players and freezing the world.
func RequestBackup(trigger string, actorEntityID string, scheduledFor *time.Time, delay time.Duration) (model.BackupJob, error) {
	if !BackupsEnabled() {
		return model.BackupJob{}, ErrBackupsDisabled
	}
	link := currentGameLink()
	if link == nil || !link.PluginConnected() {
		return model.BackupJob{}, ErrGameServerOffline
	}
	job := model.BackupJob{
		ID:                  ulid.Make().Prefixed("wbak"),
		Status:              model.BackupStatusPending,
		Trigger:             trigger,
		RequestedByEntityID: actorEntityID,
		ScheduledFor:        scheduledFor,
		DepotBucket:         config.DepotBucket,
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Checking for an active job and inserting have to be one step.
		// Manual runs carry no scheduled_for, so the unique index does not
		// cover them, and two clicks of "back up now" landing together would
		// both pass the check. The second would then fail minutes later on
		// the game server, leaving a job that only ever existed as a race.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", backupLockKey).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&model.BackupJob{}).Where("status IN ?", activeBackupStatuses).
			Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return ErrBackupInProgress
		}
		return tx.Create(&job).Error
	})
	if err != nil {
		// A second replica already claimed this cron occurrence; the unique
		// index on scheduled_for is what makes that a conflict rather than a
		// duplicate backup.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.BackupJob{}, ErrBackupInProgress
		}
		return model.BackupJob{}, err
	}

	go runBackup(job, delay)
	return job, nil
}

// backupLockKey is an arbitrary but stable id for the advisory lock that
// serializes starting a backup. Advisory locks share one namespace per
// database, so the value only has to be unlikely to collide with another
// user of the same database.
const backupLockKey int64 = 0x7761_7264_656e_01

func runBackup(job model.BackupJob, delay time.Duration) {
	if delay > 0 {
		announce(fmt.Sprintf("⚠️ **Server backup starting in %s.** You may notice a brief lag spike while the world saves.", humanDuration(delay)))
		time.Sleep(delay)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startedAt := time.Now()
	name := backupFileName(startedAt)
	upload, err := depot.InitiateUpload(ctx, job.DepotBucket, depot.InitiateRequest{
		OriginalName: name,
		Path:         backupPath(startedAt, name),
		ContentType:  BackupContentType,
		Tags: map[string]string{
			"source":  "warden",
			"kind":    "minecraft-server-backup",
			"job_id":  job.ID,
			"trigger": job.Trigger,
		},
	})
	if err != nil {
		failBackup(job.ID, fmt.Sprintf("could not reserve an upload in Depot: %v", err))
		return
	}

	job.Status = model.BackupStatusArchiving
	job.StartedAt = startedAt
	job.DepotFileID = upload.File.ID
	job.FileName = name
	if err := database.DB.Save(&job).Error; err != nil {
		failBackup(job.ID, fmt.Sprintf("could not record the upload: %v", err))
		return
	}

	link := currentGameLink()
	if link == nil {
		failBackup(job.ID, ErrGameServerOffline.Error())
		return
	}
	if err := link.StartBackup(job.ID, upload.URL, upload.Method, BackupContentType, name); err != nil {
		failBackup(job.ID, err.Error())
		return
	}
	announce("🗄️ **Server backup started.** The world is saving — expect a few seconds of lag.")
}

// BackupContentType is fixed because the presigned URL is signed with it;
// the game server must send back exactly this value on the PUT.
const BackupContentType = "application/gzip"

func backupFileName(at time.Time) string {
	return "minecraft-" + at.UTC().Format("20060102T150405Z") + ".tar.gz"
}

// backupPath sorts archives into year and month folders so Depot's
// path-prefix filter is useful once there are hundreds of them.
func backupPath(at time.Time, name string) string {
	return at.UTC().Format("backups/2006/01/") + name
}

// BackupProgress is the game server reporting a phase change mid-run.
type BackupProgress struct {
	Phase         string
	ArchiveMillis int64
	SizeBytes     int64
}

// ReportBackupProgress records that the archive is built and the upload has
// begun. Purely informational — the job's outcome comes from CompleteBackup.
func ReportBackupProgress(jobID string, progress BackupProgress) error {
	job, err := GetBackupJob(jobID)
	if err != nil {
		return err
	}
	if !job.Active() {
		return nil
	}
	if progress.Phase != model.BackupStatusUploading {
		return nil
	}
	job.Status = model.BackupStatusUploading
	job.ArchiveMillis = progress.ArchiveMillis
	job.SizeBytes = progress.SizeBytes
	return database.DB.Save(&job).Error
}

// BackupResult is the game server's final word on a job.
type BackupResult struct {
	OK            bool
	SizeBytes     int64
	ArchiveMillis int64
	UploadMillis  int64
	Error         string
}

// CompleteBackup finishes a job the game server has reported on. On success
// it asks Depot to promote the pending file, which stats the object and so
// returns the size storage actually holds rather than the size the game
// server claimed to have sent.
//
// Safe to call twice: a job that has already finished is left alone, so a
// retried report cannot double-announce or overwrite a recorded failure.
func CompleteBackup(ctx context.Context, jobID string, result BackupResult) error {
	job, err := GetBackupJob(jobID)
	if err != nil {
		return err
	}
	if !job.Active() {
		return nil
	}
	job.ArchiveMillis = result.ArchiveMillis
	job.UploadMillis = result.UploadMillis

	if !result.OK {
		message := result.Error
		if message == "" {
			message = "the game server reported a failure without a reason"
		}
		return finishBackup(job, model.BackupStatusFailed, message)
	}

	file, err := depot.CompleteUpload(ctx, job.DepotBucket, job.DepotFileID)
	if err != nil {
		// The bytes may well be in storage; what failed is Depot's record of
		// them. The job is a failure either way, because nothing can find an
		// archive that stayed PENDING.
		return finishBackup(job, model.BackupStatusFailed, fmt.Sprintf("uploaded, but Depot would not finalize the file: %v", err))
	}
	job.SizeBytes = file.SizeBytes
	return finishBackup(job, model.BackupStatusSucceeded, "")
}

func failBackup(jobID string, message string) {
	job, err := GetBackupJob(jobID)
	if err != nil {
		logger.SugarLogger.Errorf("backup %s failed (%s) and could not be loaded: %v", jobID, message, err)
		return
	}
	if !job.Active() {
		return
	}
	if err := finishBackup(job, model.BackupStatusFailed, message); err != nil {
		logger.SugarLogger.Errorf("failed to record backup failure for %s: %v", jobID, err)
	}
}

func finishBackup(job model.BackupJob, status string, message string) error {
	now := time.Now()
	job.Status = status
	job.Error = message
	job.FinishedAt = &now
	if err := database.DB.Save(&job).Error; err != nil {
		return err
	}

	switch status {
	case model.BackupStatusSucceeded:
		announce(fmt.Sprintf("✅ **Server backup complete** — %s in %s.", humanBytes(job.SizeBytes), humanDuration(backupDuration(job))))
	case model.BackupStatusTimedOut:
		announce("❌ **Server backup timed out.** The game server stopped reporting before it finished.")
	default:
		announce(fmt.Sprintf("❌ **Server backup failed.** %s", message))
	}
	logger.SugarLogger.Infof("backup %s finished: status=%s size=%d error=%q", job.ID, status, job.SizeBytes, message)
	return nil
}

func backupDuration(job model.BackupJob) time.Duration {
	if job.FinishedAt == nil || job.StartedAt.IsZero() {
		return 0
	}
	return job.FinishedAt.Sub(job.StartedAt)
}

// ---------------------------------------------------------------- scheduler

// backupTick is short relative to the warning lead, so a warning lands
// within a few seconds of its intended minute.
const backupTick = 15 * time.Second

// warningFloor suppresses a pointless warning: on a schedule that fires more
// often than the lead time, the next occurrence is always already inside the
// warning window, and players would get a notice every single cycle that
// told them nothing.
const warningFloor = 30 * time.Second

type backupRunner struct {
	// specKey detects an edit to the schedule without holding the row.
	specKey  string
	schedule cron.Schedule
	next     time.Time
	warned   bool
}

// StartBackupScheduler reaps anything the previous process left in flight,
// then runs the cron loop.
func StartBackupScheduler() {
	if !BackupsEnabled() {
		logger.SugarLogger.Infof("Server backups are disabled, scheduler not started")
		return
	}
	reapAbandonedBackups()
	go func() {
		runner := &backupRunner{}
		for {
			timeOutStaleBackups()
			runner.tick()
			time.Sleep(backupTick)
		}
	}()
	logger.SugarLogger.Infof("Started backup scheduler")
}

func (r *backupRunner) tick() {
	schedule, err := GetBackupSchedule()
	if err != nil {
		logger.SugarLogger.Errorf("backup scheduler: load schedule: %v", err)
		return
	}

	key := fmt.Sprintf("%t|%s|%s", schedule.Enabled, schedule.Cron, schedule.Timezone)
	if key != r.specKey {
		r.specKey = key
		r.schedule = nil
		r.warned = false
		if schedule.Enabled {
			parsed, err := ParseBackupCron(schedule.Cron, schedule.Timezone)
			if err != nil {
				// Only logged on change, so a bad expression saved outside the
				// API does not fill the log every tick.
				logger.SugarLogger.Errorf("backup scheduler: %v", err)
				return
			}
			r.schedule = parsed
			r.next = parsed.Next(time.Now())
			logger.SugarLogger.Infof("backup scheduler: next run %s", r.next.Format(time.RFC3339))
		}
	}
	if r.schedule == nil {
		return
	}

	now := time.Now()
	if !r.warned && !now.Before(r.next.Add(-config.BackupWarningLead)) {
		r.warned = true
		if remaining := r.next.Sub(now); remaining > warningFloor {
			announce(fmt.Sprintf("⚠️ **Scheduled server backup in %s.** You may notice a brief lag spike while the world saves.", humanDuration(remaining)))
		}
	}
	if now.Before(r.next) {
		return
	}

	occurrence := r.next
	r.next = r.schedule.Next(now)
	r.warned = false
	logger.SugarLogger.Infof("backup scheduler: firing %s, next run %s", occurrence.Format(time.RFC3339), r.next.Format(time.RFC3339))

	// Players were warned at the lead time, so the scheduled run starts
	// immediately rather than warning twice.
	if _, err := RequestBackup(model.BackupTriggerSchedule, "", &occurrence, 0); err != nil {
		if errors.Is(err, ErrBackupInProgress) {
			logger.SugarLogger.Warnf("backup scheduler: skipped %s, a backup is already running", occurrence.Format(time.RFC3339))
			return
		}
		logger.SugarLogger.Errorf("backup scheduler: could not start %s: %v", occurrence.Format(time.RFC3339), err)
		announce(fmt.Sprintf("❌ **Scheduled server backup could not start.** %s", err))
	}
}

// reapAbandonedBackups closes out jobs from a previous process. Warden holds
// the run in memory, so a restart orphans anything in flight: the game
// server's report would arrive against a job nobody is waiting on.
func reapAbandonedBackups() {
	jobs := []model.BackupJob{}
	if err := database.DB.Where("status IN ?", activeBackupStatuses).Find(&jobs).Error; err != nil {
		logger.SugarLogger.Errorf("could not reap abandoned backups: %v", err)
		return
	}
	for _, job := range jobs {
		if err := finishBackup(job, model.BackupStatusFailed, "warden restarted while this backup was running"); err != nil {
			logger.SugarLogger.Errorf("could not reap abandoned backup %s: %v", job.ID, err)
		}
	}
}

// timeOutStaleBackups gives up on a job the game server never reported the
// end of — it crashed mid-archive, or the pod was replaced. Without this a
// single lost report would block every future backup, because only one may
// be active at a time.
func timeOutStaleBackups() {
	cutoff := time.Now().Add(-config.BackupTimeout)
	jobs := []model.BackupJob{}
	err := database.DB.Where("status IN ? AND created_at < ?", activeBackupStatuses, cutoff).Find(&jobs).Error
	if err != nil {
		logger.SugarLogger.Errorf("could not time out stale backups: %v", err)
		return
	}
	for _, job := range jobs {
		if err := finishBackup(job, model.BackupStatusTimedOut, fmt.Sprintf("the game server did not report back within %s", humanDuration(config.BackupTimeout))); err != nil {
			logger.SugarLogger.Errorf("could not time out backup %s: %v", job.ID, err)
		}
	}
}

// ----------------------------------------------------------------- display

func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGT"[exp])
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	case d < time.Hour:
		d = d.Round(time.Second)
		if seconds := int(d.Seconds()) % 60; seconds != 0 {
			return fmt.Sprintf("%dm %ds", int(d.Minutes()), seconds)
		}
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		d = d.Round(time.Minute)
		if minutes := int(d.Minutes()) % 60; minutes != 0 {
			return fmt.Sprintf("%dh %dm", int(d.Hours()), minutes)
		}
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}
