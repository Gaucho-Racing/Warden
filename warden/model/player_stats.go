package model

import "time"

// Counted is one row of a per-key tally — a block mined, a mob killed, an
// item crafted. Minecraft keys these by namespaced id.
type Counted struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// StatWindow is the change in the cumulative counters over a period.
//
// Minecraft only ever reports running totals, so this cannot be read — it
// is computed by subtracting a stored snapshot. See PlayerStatsSnapshot.
type StatWindow struct {
	PlaytimeMinutes int64 `json:"playtime_minutes"`
	Deaths          int64 `json:"deaths"`
	MobKills        int64 `json:"mob_kills"`
	BlocksMined     int64 `json:"blocks_mined"`
	DistanceMeters  int64 `json:"distance_meters"`
}

// PlayerStatCounters are the scalar totals, shared by the live row and the
// historical snapshots so a delta is a field-by-field subtraction.
//
// Units are the plugin's responsibility to normalise before reporting:
//
//	PlaytimeMinutes  — Bukkit's PLAY_ONE_MINUTE is ticks despite the name,
//	                   1200 per minute. Converted plugin-side.
//	DistanceMeters   — every Bukkit distance statistic is centimetres, and
//	                   "distance" is the sum of a dozen of them (walk,
//	                   sprint, swim, boat, horse, elytra, …).
type PlayerStatCounters struct {
	PlaytimeMinutes int64 `json:"playtime_minutes"`
	Deaths          int64 `json:"deaths"`
	MobKills        int64 `json:"mob_kills"`
	PlayerKills     int64 `json:"player_kills"`
	BlocksMined     int64 `json:"blocks_mined"`
	ItemsCrafted    int64 `json:"items_crafted"`
	DistanceMeters  int64 `json:"distance_meters"`
	DamageDealt     int64 `json:"damage_dealt"`
	DamageTaken     int64 `json:"damage_taken"`
	Jumps           int64 `json:"jumps"`
	TimesSlept      int64 `json:"times_slept"`
	VillagerTrades  int64 `json:"villager_trades"`
	RaidWins        int64 `json:"raid_wins"`
	// Sessions is Bukkit's LEAVE_GAME. There is no join counter, so a
	// session count is really a "times left" count and will read one low
	// for a player who is currently online.
	Sessions int64 `json:"sessions"`
}

// PlayerStats is the live row: one per linked account, overwritten on every
// report from the plugin.
type PlayerStats struct {
	UUID     string `json:"uuid" gorm:"primaryKey"`
	Username string `json:"username"`
	PlayerStatCounters

	TopBlocks  []Counted `json:"top_blocks" gorm:"type:jsonb;serializer:json"`
	TopMobs    []Counted `json:"top_mobs" gorm:"type:jsonb;serializer:json"`
	TopCrafted []Counted `json:"top_crafted" gorm:"type:jsonb;serializer:json"`

	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	ReportedAt time.Time `json:"reported_at"`

	// Source is "plugin" for a real report and "none" when the account is
	// linked but has never been reported on, so the portal can say "play
	// once and these will appear" rather than rendering a wall of zeroes
	// as though they were real.
	Source string `json:"source" gorm:"-"`

	Last7Days *StatWindow `json:"last_7_days,omitempty" gorm:"-"`
}

func (PlayerStats) TableName() string {
	return "warden_player_stats"
}

const (
	StatSourcePlugin = "plugin"
	StatSourceNone   = "none"
)

// PlayerStatsSnapshot is one player's counters as of one day.
//
// Written on the first report of each UTC day rather than by a scheduler:
// a snapshot only has to exist for days somebody actually played, and
// tying it to ingest means there is no cron to fail silently.
type PlayerStatsSnapshot struct {
	UUID string `json:"uuid" gorm:"primaryKey"`
	// Day is midnight UTC of the day the snapshot covers.
	Day time.Time `json:"day" gorm:"primaryKey"`
	PlayerStatCounters
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (PlayerStatsSnapshot) TableName() string {
	return "warden_player_stats_snapshot"
}
