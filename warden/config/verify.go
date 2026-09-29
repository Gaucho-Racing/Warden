package config

import (
	"os"
	"time"

	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

const (
	defaultLinkTokenTTL      = 15 * time.Minute
	defaultGroupSyncInterval = 60 * time.Second
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
