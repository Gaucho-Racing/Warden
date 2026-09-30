package bridge

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gaucho-racing/warden/warden/service"
)

const nameTTL = 5 * time.Minute

// names resolves who someone is on the other side of the bridge, cached
// because every chat line would otherwise cost a database read and one or two
// Sentinel calls.
type names struct {
	mu      sync.Mutex
	players map[string]cachedName
	discord map[string]cachedName
}

type cachedName struct {
	name    string
	expires time.Time
}

func newNames() *names {
	return &names{players: map[string]cachedName{}, discord: map[string]cachedName{}}
}

// player is "First (username)" for a linked player, else the Minecraft name.
func (n *names) player(ctx context.Context, uuid string) string {
	if name, ok := n.lookup(n.players, uuid); ok {
		return name
	}
	name := uuid
	if account, err := service.GetAccountByUUID(uuid); err == nil {
		name = withFirstName(service.FirstName(ctx, account.EntityID), account.Username)
	}
	n.store(n.players, uuid, name)
	return name
}

// discordAuthor is "First (username)" for a member with a linked Minecraft
// account, their Sentinel first name if they are in Sentinel but not linked,
// and fallback (their Discord name) if Sentinel does not know them.
func (n *names) discordAuthor(ctx context.Context, discordID string, fallback string) string {
	if name, ok := n.lookup(n.discord, discordID); ok {
		return name
	}
	name := fallback
	entityID, err := sentinel.GetEntityIDByExternalAuth(ctx, config.SentinelSAToken, "DISCORD", discordID)
	var sentinelErr sentinel.Error
	switch {
	case err == nil:
		first := service.FirstName(ctx, entityID)
		if account, err := service.GetAccountByEntityID(entityID); err == nil {
			name = withFirstName(first, account.Username)
		} else if first != "" {
			name = first
		}
	case errors.As(err, &sentinelErr) && sentinelErr.Code == http.StatusNotFound:
	default:
		// Don't cache a Sentinel outage as "not in Sentinel".
		logger.SugarLogger.Warnf("bridge: resolve discord user %s: %v", discordID, err)
		return name
	}
	n.store(n.discord, discordID, name)
	return name
}

func withFirstName(first string, username string) string {
	if first == "" {
		return username
	}
	return first + " (" + username + ")"
}

func (n *names) lookup(cache map[string]cachedName, key string) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	entry, ok := cache[key]
	if !ok || time.Now().After(entry.expires) {
		return "", false
	}
	return entry.name, true
}

func (n *names) store(cache map[string]cachedName, key string, name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	cache[key] = cachedName{name: name, expires: time.Now().Add(nameTTL)}
}
