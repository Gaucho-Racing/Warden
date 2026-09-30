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
	name     string
	username string
	expires  time.Time
}

func newNames() *names {
	return &names{players: map[string]cachedName{}, discord: map[string]cachedName{}}
}

// player is "First (username)" for a linked player, else the Minecraft name.
func (n *names) player(ctx context.Context, uuid string) string {
	if entry, ok := n.lookup(n.players, uuid); ok {
		return entry.name
	}
	name := uuid
	if account, err := service.GetAccountByUUID(uuid); err == nil {
		name = withFirstName(service.FirstName(ctx, account.EntityID), account.Username)
	}
	n.store(n.players, uuid, cachedName{name: name})
	return name
}

// discordAuthor names a Discord member for the game. Name is their Sentinel
// first name, or fallback (their Discord name) if Sentinel does not know
// them. Username is their Minecraft name when they have a linked account and
// empty otherwise; it is kept separate so the plugin can style the two parts
// differently without parsing a string that might contain parentheses.
func (n *names) discordAuthor(ctx context.Context, discordID string, fallback string) (name string, username string) {
	if entry, ok := n.lookup(n.discord, discordID); ok {
		return entry.name, entry.username
	}
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
		// Don't cache a Sentinel outage as "not in Sentinel".
		logger.SugarLogger.Warnf("bridge: resolve discord user %s: %v", discordID, err)
		return name, ""
	}
	n.store(n.discord, discordID, cachedName{name: name, username: username})
	return name, username
}

func withFirstName(first string, username string) string {
	if first == "" {
		return username
	}
	return first + " (" + username + ")"
}

func (n *names) lookup(cache map[string]cachedName, key string) (cachedName, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	entry, ok := cache[key]
	if !ok || time.Now().After(entry.expires) {
		return cachedName{}, false
	}
	return entry, true
}

func (n *names) store(cache map[string]cachedName, key string, entry cachedName) {
	n.mu.Lock()
	defer n.mu.Unlock()
	entry.expires = time.Now().Add(nameTTL)
	cache[key] = entry
}
