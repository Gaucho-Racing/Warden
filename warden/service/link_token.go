package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gaucho-racing/ulid-go"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"gorm.io/gorm"
)

var (
	ErrLinkTokenNotFound   = errors.New("link token not found")
	ErrLinkTokenExpired    = errors.New("link token has expired")
	ErrLinkTokenConsumed   = errors.New("link token has already been used")
	ErrUUIDAlreadyLinked   = errors.New("this minecraft account is already linked")
	ErrEntityAlreadyLinked = errors.New("your sentinel account already has a minecraft account linked")
)

func GetLinkToken(id string) (model.LinkToken, error) {
	var token model.LinkToken
	if err := database.DB.Where("id = ?", id).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.LinkToken{}, ErrLinkTokenNotFound
		}
		return model.LinkToken{}, err
	}
	return token, nil
}

// CreateLinkToken mints a pending link for an already-Mojang-authenticated
// UUID. Any unconsumed token for the same UUID is invalidated first so a
// player spamming the join flow can't leave a trail of live tokens behind.
func CreateLinkToken(uuid string, username string) (model.LinkToken, error) {
	if _, err := GetAccountByUUID(uuid); err == nil {
		return model.LinkToken{}, ErrUUIDAlreadyLinked
	} else if !errors.Is(err, ErrAccountNotFound) {
		return model.LinkToken{}, err
	}

	now := time.Now()
	if err := database.DB.Where("uuid = ? AND consumed_at IS NULL", uuid).
		Delete(&model.LinkToken{}).Error; err != nil {
		return model.LinkToken{}, err
	}

	token := model.LinkToken{
		ID:        ulid.Make().Prefixed("lnk"),
		UUID:      uuid,
		Username:  username,
		ExpiresAt: now.Add(config.LinkTokenTTL),
	}
	if err := database.DB.Create(&token).Error; err != nil {
		return model.LinkToken{}, err
	}
	return token, nil
}

// ConsumeLinkToken binds a pending token's UUID to the Sentinel entity that
// opened it in the browser. Both halves are authenticated before this point:
// Mojang vouched for the UUID at join, Sentinel vouched for the entity at
// login. This is the only place the two are joined.
func ConsumeLinkToken(id string, entityID string) (model.MinecraftAccount, error) {
	token, err := GetLinkToken(id)
	if err != nil {
		return model.MinecraftAccount{}, err
	}
	now := time.Now()
	if token.IsConsumed() {
		return model.MinecraftAccount{}, ErrLinkTokenConsumed
	}
	if token.IsExpired(now) {
		return model.MinecraftAccount{}, ErrLinkTokenExpired
	}

	if _, err := GetAccountByUUID(token.UUID); err == nil {
		return model.MinecraftAccount{}, ErrUUIDAlreadyLinked
	} else if !errors.Is(err, ErrAccountNotFound) {
		return model.MinecraftAccount{}, err
	}
	if _, err := GetAccountByEntityID(entityID); err == nil {
		return model.MinecraftAccount{}, ErrEntityAlreadyLinked
	} else if !errors.Is(err, ErrAccountNotFound) {
		return model.MinecraftAccount{}, err
	}

	account := model.MinecraftAccount{
		UUID:             token.UUID,
		EntityID:         entityID,
		Username:         token.Username,
		LinkedViaTokenID: token.ID,
		LastSeenAt:       now,
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		return tx.Model(&model.LinkToken{}).
			Where("id = ? AND consumed_at IS NULL", token.ID).
			Updates(map[string]any{"consumed_at": now, "entity_id": entityID}).Error
	})
	if err != nil {
		return model.MinecraftAccount{}, err
	}
	return account, nil
}

// LinkURL is the clickable URL the plugin puts in chat. Warden builds it so
// the portal's origin can change without reshipping the plugin jar.
func LinkURL(token model.LinkToken) string {
	return fmt.Sprintf("%s/link/%s", strings.TrimRight(config.PublicBaseURL, "/"), token.ID)
}

// PurgeExpiredLinkTokens drops tokens that can no longer be used. Consumed
// tokens are kept — MinecraftAccount.LinkedViaTokenID references them for
// audit.
func PurgeExpiredLinkTokens() error {
	return database.DB.Where("consumed_at IS NULL AND expires_at < ?", time.Now()).
		Delete(&model.LinkToken{}).Error
}
