package model

import "time"

// ServerStatus is one sample of the game server's health, reported by the
// plugin every minute and after joins and quits. Kept as history for
// analytics, and the latest one drives the Discord presence and topic.
type ServerStatus struct {
	RecordedAt    time.Time `json:"recorded_at" gorm:"primaryKey"`
	Online        int       `json:"online"`
	MaxPlayers    int       `json:"max_players"`
	UniquePlayers int       `json:"unique_players"`
	TPS           float64   `json:"tps"`
	MSPT          float64   `json:"mspt"`
	// When the game server process started; uptime is RecordedAt minus this.
	StartedAt time.Time `json:"started_at"`
}

func (ServerStatus) TableName() string {
	return "warden_server_status"
}
