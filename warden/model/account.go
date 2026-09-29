package model

import (
	"time"

	"gorm.io/gorm"
)

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

	// Derived, never stored — see AvatarURL.
	AvatarURL string    `json:"avatar_url" gorm:"-"`
	Identity  *Identity `json:"identity,omitempty" gorm:"-"`
}

func (MinecraftAccount) TableName() string {
	return "warden_minecraft_account"
}

// AfterFind fills the rendered-head URL on every read so callers never have
// to remember to, and so the address exists in exactly one place rather than
// being rebuilt in the frontend from a UUID.
func (a *MinecraftAccount) AfterFind(tx *gorm.DB) error {
	a.AvatarURL = AvatarURL(a.UUID)
	return nil
}

// AvatarURL returns a rendered head for a UUID.
//
// Crafatar rather than mc-heads: it documents its caching (20 minute skin
// refresh, 60 minute browser cache) and supports CORS, where mc-heads
// documents neither. Both are third parties that see our members' UUIDs —
// self-hosting the crop from textures.minecraft.net is the only way to stop
// that, and is the obvious next step if it ever matters.
//
// overlay includes the skin's second layer, which most modern skins use for
// hair and hats; without it a lot of players render bald.
func AvatarURL(uuid string) string {
	return "https://crafatar.com/avatars/" + uuid + "?size=128&overlay"
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
