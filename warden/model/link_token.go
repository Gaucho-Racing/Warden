package model

import "time"

// LinkToken is a short-lived, single-use grant that lets whoever opens it in
// a browser claim a specific Minecraft UUID for their own Sentinel entity.
//
// The plugin mints one on behalf of a player who has already been
// authenticated by Mojang, so the UUID side is trustworthy before the token
// exists. The browser side is authenticated by Sentinel SSO. The token is
// only the thing that carries the UUID across from one to the other, which
// is why it is unguessable and expires fast.
type LinkToken struct {
	ID         string     `json:"id" gorm:"primaryKey"`
	UUID       string     `json:"uuid" gorm:"index"`
	Username   string     `json:"username"`
	EntityID   string     `json:"entity_id" gorm:"index"`
	ConsumedAt *time.Time `json:"consumed_at"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"index"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

func (LinkToken) TableName() string {
	return "warden_link_token"
}

func (t LinkToken) IsConsumed() bool {
	return t.ConsumedAt != nil
}

func (t LinkToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

func (t LinkToken) IsUsable(now time.Time) bool {
	return !t.IsConsumed() && !t.IsExpired(now)
}
