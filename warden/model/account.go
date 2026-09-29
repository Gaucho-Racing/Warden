package model

import "time"

// MinecraftAccount binds an authenticated Minecraft UUID to a Sentinel
// entity. The UUID is the primary key because it is the only stable
// identifier Mojang gives us — usernames are mutable and get reused.
//
// Warden owns this table outright. Sentinel has no notion of a Minecraft
// account, and Warden has no scope to teach it one.
type MinecraftAccount struct {
	UUID             string    `json:"uuid" gorm:"primaryKey"`
	EntityID         string    `json:"entity_id" gorm:"uniqueIndex"`
	Username         string    `json:"username" gorm:"index"`
	LinkedViaTokenID string    `json:"linked_via_token_id"`
	LastSeenAt       time.Time `json:"last_seen_at"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Identity *Identity `json:"identity,omitempty" gorm:"-"`
}

func (MinecraftAccount) TableName() string {
	return "warden_minecraft_account"
}

// Identity is the hydrated Sentinel-side view of an entity, filled in from
// the Sentinel API at response time. Never persisted — Sentinel stays the
// source of truth for names and avatars.
type Identity struct {
	EntityID  string `json:"entity_id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}
