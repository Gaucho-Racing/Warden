package model

import "time"

// BlockCount is one entry of Minecraft's per-block mined tally.
type BlockCount struct {
	Block string `json:"block"`
	Count int    `json:"count"`
}

// StatWindow is the change in the cumulative counters over a period.
// Minecraft only ever reports running totals, so a delta requires Warden to
// snapshot the counters on a schedule and subtract — it cannot be derived
// from a single read. Nothing snapshots yet; see service.MockPlayerStats.
type StatWindow struct {
	PlaytimeMinutes int `json:"playtime_minutes"`
	Deaths          int `json:"deaths"`
	MobKills        int `json:"mob_kills"`
	BlocksMined     int `json:"blocks_mined"`
}

// PlayerStats is the shape the portal renders. It mirrors the counters the
// server already keeps in world/stats/<uuid>.json:
//
//	minecraft:custom  -> play_time (ticks), deaths, mob_kills, walk_one_cm, …
//	minecraft:mined   -> per-block counts
//
// The real implementation is the plugin reading those files (or the Bukkit
// Statistic API, which is the same data live) and POSTing them to Warden on
// a timer and on quit. Source says where a given payload came from so the UI
// can be honest about it.
type PlayerStats struct {
	UUID            string       `json:"uuid"`
	Username        string       `json:"username"`
	Source          string       `json:"source"`
	PlaytimeMinutes int          `json:"playtime_minutes"`
	Deaths          int          `json:"deaths"`
	MobKills        int          `json:"mob_kills"`
	BlocksMined     int          `json:"blocks_mined"`
	DistanceMeters  int          `json:"distance_meters"`
	JoinCount       int          `json:"join_count"`
	FirstSeen       time.Time    `json:"first_seen"`
	LastSeen        time.Time    `json:"last_seen"`
	TopBlocks       []BlockCount `json:"top_blocks"`
	Last7Days       *StatWindow  `json:"last_7_days,omitempty"`
}

const (
	StatSourceMock   = "mock"
	StatSourcePlugin = "plugin"
)
