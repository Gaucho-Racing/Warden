package config

import (
	"os"
	"time"

	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

const (
	defaultLinkTokenTTL      = 15 * time.Minute
	defaultGroupSyncInterval = 60 * time.Second
	defaultBackupWarningLead = 5 * time.Minute
	defaultBackupManualDelay = 15 * time.Second
	defaultBackupTimeout     = 60 * time.Minute
	defaultBackupTimezone    = "America/Los_Angeles"
)

func Verify() {
	if Env == "" {
		Env = "PROD"
		logger.SugarLogger.Infof("ENV is not set, defaulting to %s", Env)
	}
	if Port == "" {
		Port = "9999"
		logger.SugarLogger.Infof("PORT is not set, defaulting to %s", Port)
	}
	if DatabaseHost == "" {
		DatabaseHost = "localhost"
		logger.SugarLogger.Infof("DATABASE_HOST is not set, defaulting to %s", DatabaseHost)
	}
	if DatabasePort == "" {
		DatabasePort = "5432"
		logger.SugarLogger.Infof("DATABASE_PORT is not set, defaulting to %s", DatabasePort)
	}
	if DatabaseUser == "" {
		DatabaseUser = "postgres"
		logger.SugarLogger.Infof("DATABASE_USER is not set, defaulting to %s", DatabaseUser)
	}
	if DatabasePassword == "" {
		DatabasePassword = "password"
		logger.SugarLogger.Infof("DATABASE_PASSWORD is not set, defaulting to %s", DatabasePassword)
	}
	if DatabaseName == "" {
		DatabaseName = "warden"
		logger.SugarLogger.Infof("DATABASE_NAME is not set, defaulting to %s", DatabaseName)
	}
	if SentinelURL == "" {
		logger.SugarLogger.Fatal("SENTINEL_URL is required")
	}
	if SentinelClientID == "" {
		logger.SugarLogger.Fatal("SENTINEL_CLIENT_ID is required")
	}
	if SentinelClientSecret == "" {
		logger.SugarLogger.Fatal("SENTINEL_CLIENT_SECRET is required")
	}
	// Without a plugin token every /plugin/* route would be unauthenticated,
	// which would let anyone on the network mint link tokens for a UUID they
	// don't own. Refuse to boot rather than run wide open.
	if PluginToken == "" {
		logger.SugarLogger.Fatal("PLUGIN_TOKEN is required")
	}
	if PublicBaseURL == "" {
		logger.SugarLogger.Fatal("PUBLIC_BASE_URL is required")
	}
	LinkTokenTTL = durationOrDefault("LINK_TOKEN_TTL", defaultLinkTokenTTL)
	GroupSyncInterval = durationOrDefault("GROUP_SYNC_INTERVAL", defaultGroupSyncInterval)
	verifyBackups()
}

// verifyBackups never fails the boot. Backups are an add-on to running the
// game server, so an unconfigured or misconfigured Depot leaves the feature
// off and everything else working, rather than taking the service down.
func verifyBackups() {
	BackupWarningLead = durationOrDefault("BACKUP_WARNING_LEAD", defaultBackupWarningLead)
	BackupManualDelay = durationOrDefault("BACKUP_MANUAL_DELAY", defaultBackupManualDelay)
	BackupTimeout = durationOrDefault("BACKUP_TIMEOUT", defaultBackupTimeout)

	if BackupTimezone == "" {
		BackupTimezone = defaultBackupTimezone
	}
	if _, err := time.LoadLocation(BackupTimezone); err != nil {
		logger.SugarLogger.Warnf("BACKUP_TIMEZONE %q is not a known IANA zone (%v), defaulting to %s", BackupTimezone, err, defaultBackupTimezone)
		BackupTimezone = defaultBackupTimezone
	}

	// The service account is the only credential the backup path needs; the
	// Depot origin and bucket are fixed.
	if SentinelSAToken == "" {
		logger.SugarLogger.Warnf("SENTINEL_SA_TOKEN is not set, server backups are disabled")
		return
	}
	logger.SugarLogger.Infof("Server backups upload to %s bucket %q", DepotURL, DepotBucket)
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		logger.SugarLogger.Infof("%s is not set, defaulting to %s", key, fallback)
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		logger.SugarLogger.Warnf("%s is not a valid duration (%q), defaulting to %s", key, raw, fallback)
		return fallback
	}
	return parsed
}
