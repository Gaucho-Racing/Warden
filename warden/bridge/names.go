package bridge

import (
	"context"
	"errors"
	"net/http"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gaucho-racing/warden/warden/service"
)

// Names are resolved on every message rather than cached, so linking,
// unlinking and Sentinel name changes show up on the next message. At chat
// volume that is one database read plus one or two Sentinel calls per line,
// bounded by resolveTimeout if Sentinel is slow.

// playerName is "First (username)" for a linked player, else fallback, the
// Minecraft name the plugin sent.
func playerName(ctx context.Context, uuid string, fallback string) string {
	account, err := service.GetAccountByUUID(uuid)
	if err != nil {
		if fallback == "" {
			return uuid
		}
		return fallback
	}
	return withFirstName(service.FirstName(ctx, account.EntityID), account.Username)
}

// discordAuthor names a Discord member for the game. Name is their Sentinel
// first name, or fallback (their Discord name) if Sentinel does not know
// them. Username is their Minecraft name when they have a linked account and
// empty otherwise; it is kept separate so the plugin can style the two parts
// differently without parsing a string that might contain parentheses.
func discordAuthor(ctx context.Context, discordID string, fallback string) (name string, username string) {
	name = fallback
	entityID, err := sentinel.GetEntityIDByExternalAuth(ctx, config.SentinelSAToken, "DISCORD", discordID)
	var sentinelErr sentinel.Error
	switch {
	case err == nil:
		if first := service.FirstName(ctx, entityID); first != "" {
			name = first
		}
		if account, err := service.GetAccountByEntityID(entityID); err == nil {
			username = account.Username
		}
	case errors.As(err, &sentinelErr) && sentinelErr.Code == http.StatusNotFound:
	default:
		logger.SugarLogger.Warnf("bridge: resolve discord user %s: %v", discordID, err)
	}
	return name, username
}

func withFirstName(first string, username string) string {
	if first == "" {
		return username
	}
	return first + " (" + username + ")"
}
