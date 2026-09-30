package config

import (
	"os"
	"time"
)

const Name = "warden"
const Version = "1.7.0"

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

var LinkTokenTTL time.Duration
var GroupSyncInterval time.Duration

func IsProduction() bool {
	return Env == "PROD"
}
