package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/depot"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type backupScheduleResponse struct {
	Cron              string      `json:"cron"`
	Timezone          string      `json:"timezone"`
	Enabled           bool        `json:"enabled"`
	UpdatedByEntityID string      `json:"updated_by_entity_id"`
	UpdatedAt         time.Time   `json:"updated_at"`
	NextRuns          []time.Time `json:"next_runs"`
	// Error explains why NextRuns is empty for a schedule that is on. An
	// expression can only become invalid by being written straight to the
	// database, but the portal should say so rather than show nothing.
	Error string `json:"error,omitempty"`
}

type backupStatusResponse struct {
	Enabled  bool                   `json:"enabled"`
	Bucket   string                 `json:"bucket"`
	Schedule backupScheduleResponse `json:"schedule"`
	// ServerConnected is whether a backup could start right now. The portal
	// disables the button on it, because a backup is the game server's work.
	ServerConnected bool              `json:"server_connected"`
	Active          *model.BackupJob  `json:"active,omitempty"`
	Last            *model.BackupJob  `json:"last,omitempty"`
	Jobs            []model.BackupJob `json:"jobs"`
}

// GetBackupStatus is everything the backups page needs in one request: the
// schedule and its upcoming runs, whether a backup is in flight, and recent
// history. Visible to every signed-in member — knowing the world is backed
// up is not privileged information; starting one is.
func GetBackupStatus(c *gin.Context) {
	Require(c, RequestTokenExists(c))

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if err != nil || limit < 1 || limit > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer between 1 and 200"})
		return
	}

	schedule, err := service.GetBackupSchedule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	jobs, err := service.ListBackupJobs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	active, err := service.ActiveBackupJob()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	last, err := service.LastFinishedBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backupStatusResponse{
		Enabled:         service.BackupsEnabled(),
		Bucket:          config.DepotBucket,
		Schedule:        describeSchedule(schedule),
		ServerConnected: service.GameServerConnected(),
		Active:          active,
		Last:            last,
		Jobs:            jobs,
	})
}

// GetBackupSchedule is the schedule on its own, for anything that does not
// need the whole backups page.
func GetBackupSchedule(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	schedule, err := service.GetBackupSchedule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, describeSchedule(schedule))
}

type saveBackupScheduleRequest struct {
	Cron     string `json:"cron" binding:"required"`
	Timezone string `json:"timezone"`
	// Pointer so an omitted field keeps the current setting rather than
	// silently turning scheduled backups off.
	Enabled *bool `json:"enabled"`
}

func UpdateBackupSchedule(c *gin.Context) {
	Require(c, RequestUserIsMinecraftAdmin(c))

	var req saveBackupScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	current, err := service.GetBackupSchedule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	enabled := current.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	saved, err := service.SaveBackupSchedule(req.Cron, req.Timezone, enabled, GetRequestTokenEntityID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBackupScheduleUpdated,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        saved.ID,
		Detail:          scheduleDetail(saved),
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.JSON(http.StatusOK, describeSchedule(saved))
}

type schedulePreviewResponse struct {
	Cron     string      `json:"cron"`
	Timezone string      `json:"timezone"`
	Valid    bool        `json:"valid"`
	Error    string      `json:"error,omitempty"`
	NextRuns []time.Time `json:"next_runs"`
}

// PreviewBackupSchedule answers "when would this expression actually run".
// The portal calls it as the field is edited, so an invalid expression is a
// 200 with valid=false rather than a 400 — a half-typed cron is not a
// client error, it is the normal state of a text field.
//
// This is deliberately server-side. A cron parser in the browser would be a
// second implementation of the semantics the scheduler uses, and the first
// time the two disagreed the preview would be confidently wrong.
func PreviewBackupSchedule(c *gin.Context) {
	Require(c, RequestTokenExists(c))

	spec := c.Query("cron")
	timezone := c.DefaultQuery("timezone", config.BackupTimezone)
	count, err := strconv.Atoi(c.DefaultQuery("count", "3"))
	if err != nil || count < 1 || count > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be an integer between 1 and 10"})
		return
	}

	response := schedulePreviewResponse{Cron: spec, Timezone: timezone, NextRuns: []time.Time{}}
	runs, err := service.NextBackupRuns(spec, timezone, count)
	if err != nil {
		response.Error = err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	response.Valid = true
	response.NextRuns = runs
	c.JSON(http.StatusOK, response)
}

// CreateBackup is the "back up now" button. Restricted to MinecraftAdmins:
// it freezes the world mid-session and writes a multi-gigabyte archive to
// the game server's disk.
//
// Answers 202, not 201 — the job exists, but the work has not started and
// will not finish for minutes.
func CreateBackup(c *gin.Context) {
	Require(c, RequestUserIsMinecraftAdmin(c))

	job, err := service.RequestBackup(model.BackupTriggerManual, GetRequestTokenEntityID(c), nil, config.BackupManualDelay)
	switch {
	case errors.Is(err, service.ErrBackupsDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	case errors.Is(err, service.ErrGameServerOffline):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	case errors.Is(err, service.ErrBackupInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBackupStarted,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        job.ID,
		Detail:          "manual backup requested",
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.JSON(http.StatusAccepted, job)
}

// CreateBackupDownloadURL mints a short-lived Depot link to a finished
// archive. MinecraftAdmins only, and for a stronger reason than the button:
// a world archive holds every player's inventory, every chest and every
// sign on the map, and the link Depot returns is bearer authority over it.
func CreateBackupDownloadURL(c *gin.Context) {
	Require(c, RequestUserIsMinecraftAdmin(c))

	job, err := service.GetBackupJob(c.Param("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if job.Status != model.BackupStatusSucceeded || job.DepotFileID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "this backup did not complete, so there is nothing to download"})
		return
	}

	download, err := depot.CreateDownloadURL(c.Request.Context(), job.DepotBucket, job.DepotFileID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBackupDownloaded,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        job.ID,
		Detail:          job.FileName,
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, download)
}

// ------------------------------------------------------------ plugin realm

type backupProgressRequest struct {
	Phase         string `json:"phase" binding:"required"`
	ArchiveMillis int64  `json:"archive_millis"`
	SizeBytes     int64  `json:"size_bytes"`
}

// ReportBackupProgress is the game server saying the archive is built and
// the upload has started, so the portal can show the phase rather than a
// single opaque "running" for several minutes.
func ReportBackupProgress(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	var req backupProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := service.ReportBackupProgress(c.Param("id"), service.BackupProgress{
		Phase:         req.Phase,
		ArchiveMillis: req.ArchiveMillis,
		SizeBytes:     req.SizeBytes,
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type backupCompleteRequest struct {
	OK            bool   `json:"ok"`
	SizeBytes     int64  `json:"size_bytes"`
	ArchiveMillis int64  `json:"archive_millis"`
	UploadMillis  int64  `json:"upload_millis"`
	Error         string `json:"error"`
}

// CompleteBackup is the game server's final report. Warden verifies the
// object with Depot rather than taking the reported size on trust, so a
// plugin that lies about success still produces a failed job.
func CompleteBackup(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	var req backupCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := service.CompleteBackup(c.Request.Context(), c.Param("id"), service.BackupResult{
		OK:            req.OK,
		SizeBytes:     req.SizeBytes,
		ArchiveMillis: req.ArchiveMillis,
		UploadMillis:  req.UploadMillis,
		Error:         req.Error,
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func describeSchedule(schedule model.BackupSchedule) backupScheduleResponse {
	response := backupScheduleResponse{
		Cron:              schedule.Cron,
		Timezone:          schedule.Timezone,
		Enabled:           schedule.Enabled,
		UpdatedByEntityID: schedule.UpdatedByEntityID,
		UpdatedAt:         schedule.UpdatedAt,
		NextRuns:          []time.Time{},
	}
	if !schedule.Enabled {
		return response
	}
	runs, err := service.NextBackupRuns(schedule.Cron, schedule.Timezone, 3)
	if err != nil {
		response.Error = err.Error()
		return response
	}
	response.NextRuns = runs
	return response
}

func scheduleDetail(schedule model.BackupSchedule) string {
	if !schedule.Enabled {
		return "scheduled backups disabled"
	}
	return schedule.Cron + " " + schedule.Timezone
}
