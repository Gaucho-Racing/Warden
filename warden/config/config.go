package config

import (
	"os"
	"time"
)

const Name = "warden"
const Version = "1.10.0"

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}

var Env = os.Getenv("ENV")
var Port = os.Getenv("PORT")

var DatabaseHost = os.Getenv("DATABASE_HOST")
var DatabasePort = os.Getenv("DATABASE_PORT")
var DatabaseUser = os.Getenv("DATABASE_USER")
var DatabasePassword = os.Getenv("DATABASE_PASSWORD")
var DatabaseName = os.Getenv("DATABASE_NAME")

var SentinelURL = os.Getenv("SENTINEL_URL")
var SentinelClientID = os.Getenv("SENTINEL_CLIENT_ID")
var SentinelClientSecret = os.Getenv("SENTINEL_CLIENT_SECRET")
var SentinelSAToken = os.Getenv("SENTINEL_SA_TOKEN")
var SentinelRedirectURI = os.Getenv("SENTINEL_REDIRECT_URI")

// PluginToken is the bearer the Minecraft plugin presents on /plugin/*. It
// lives in a config file on the game server, so it is deliberately scoped to
// Warden alone — it is never a Sentinel credential and can never read or
// write Sentinel state directly.
var PluginToken = os.Getenv("PLUGIN_TOKEN")

// PublicBaseURL is the origin players reach the link portal on. The plugin
// never builds link URLs itself; Warden hands it a fully-formed URL so the
// portal can move without reshipping a jar.
var PublicBaseURL = os.Getenv("PUBLIC_BASE_URL")

// The Discord bridge runs only when both are set. The bot token never leaves
// this service; the plugin only ever sees game events and rendered names.
var DiscordToken = os.Getenv("DISCORD_TOKEN")
var DiscordChannelID = os.Getenv("DISCORD_CHANNEL_ID")

func DiscordBridgeEnabled() bool {
	return DiscordToken != "" && DiscordChannelID != ""
}

// Depot is where server backups land. There is one Depot and one bucket for
// this, both fixed, so they are constants rather than configuration — the
// bucket's write grant names the Warden application specifically, which
// makes pointing at a different one a Depot-side change anyway.
//
// Warden authenticates with the same Sentinel service-account token it uses
// for group reads.
const DepotURL = "https://depot.gauchoracing.com"
const DepotBucket = "minecraft"

// BackupTimezone is the IANA zone cron expressions are evaluated in. Without
// it a schedule would silently shift twice a year, because the pod's clock
// is UTC and "2am" means local time to whoever wrote the expression.
var BackupTimezone = os.Getenv("BACKUP_TIMEZONE")

var LinkTokenTTL time.Duration
var GroupSyncInterval time.Duration

// BackupWarningLead is how far ahead of a scheduled backup players are
// warned. BackupManualDelay is the shorter grace period on a manual run,
// where somebody is waiting on the button. BackupTimeout gives up on a job
// the game server never reported the end of.
var BackupWarningLead time.Duration
var BackupManualDelay time.Duration
var BackupTimeout time.Duration

func IsProduction() bool {
	return Env == "PROD"
}
