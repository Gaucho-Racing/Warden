package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
)

// ResolvedPermissions is what the plugin applies to a player.
//
// SentinelGroups lists only the groups that matched a binding, not every
// group the member belongs to — see applyBindings. It is a
// complete desired state, not a delta: the plugin sets the managed LuckPerms
// groups to exactly LuckPermsGroups and leaves every unmanaged group alone.
// Sending the full set means a dropped update can never leave a player
// holding a permission that was revoked.
type ResolvedPermissions struct {
	UUID            string    `json:"uuid"`
	Username        string    `json:"username"`
	Linked          bool      `json:"linked"`
	EntityID        string    `json:"entity_id,omitempty"`
	DisplayName     string    `json:"display_name,omitempty"`
	SentinelGroups  []string  `json:"sentinel_groups"`
	LuckPermsGroups []string  `json:"luckperms_groups"`
	Permissions     []string  `json:"permissions"`
	ResolvedAt      time.Time `json:"resolved_at"`
}

// ResolveForUUID computes a player's desired Minecraft permissions from
// their current Sentinel group membership.
//
// One Sentinel call per resolve, which is the right shape for a join: the
// player is already waiting, and the alternative (a full reverse index) is
// far more work for a single answer. The bulk sweep uses ResolveAll instead.
func ResolveForUUID(ctx context.Context, uuid string, username string) (ResolvedPermissions, error) {
	resolved := ResolvedPermissions{
		UUID:            uuid,
		Username:        username,
		SentinelGroups:  []string{},
		LuckPermsGroups: []string{},
		Permissions:     []string{},
		ResolvedAt:      time.Now(),
	}

	account, err := GetAccountByUUID(uuid)
	if errors.Is(err, ErrAccountNotFound) {
		return resolved, nil
	}
	if err != nil {
		return resolved, err
	}
	resolved.Linked = true
	resolved.EntityID = account.EntityID
	if resolved.Username == "" {
		resolved.Username = account.Username
	}

	groups, err := sentinel.GetEntityGroups(ctx, config.SentinelSAToken, account.EntityID)
	if err != nil {
		return resolved, err
	}
	bindings, err := ListBindings()
	if err != nil {
		return resolved, err
	}

	applyBindings(&resolved, groups, bindings)
	resolved.DisplayName = displayNames(ctx, []string{account.EntityID})[account.EntityID]
	return resolved, nil
}

// ResolveAll computes desired permissions for every linked account. It walks
// bound groups rather than players — one Sentinel call per binding instead
// of one per player, which wins as soon as there are more players than
// bindings, and there always are.
func ResolveAll(ctx context.Context) ([]ResolvedPermissions, error) {
	accounts, err := ListAccounts()
	if err != nil {
		return nil, err
	}
	bindings, err := ListBindings()
	if err != nil {
		return nil, err
	}

	// entityID -> group IDs, built from the member list of each bound group.
	groupsByEntity := make(map[string][]sentinel.Group)
	for _, binding := range bindings {
		members, err := sentinel.GetGroupMembers(ctx, config.SentinelSAToken, binding.GroupID)
		if err != nil {
			return nil, err
		}
		group := sentinel.Group{ID: binding.GroupID, Name: binding.GroupName}
		for _, member := range members {
			groupsByEntity[member.EntityID] = append(groupsByEntity[member.EntityID], group)
		}
	}

	entityIDs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		entityIDs = append(entityIDs, account.EntityID)
	}
	names := displayNames(ctx, entityIDs)

	now := time.Now()
	results := make([]ResolvedPermissions, 0, len(accounts))
	for _, account := range accounts {
		resolved := ResolvedPermissions{
			UUID:            account.UUID,
			Username:        account.Username,
			Linked:          true,
			EntityID:        account.EntityID,
			DisplayName:     names[account.EntityID],
			SentinelGroups:  []string{},
			LuckPermsGroups: []string{},
			Permissions:     []string{},
			ResolvedAt:      now,
		}
		applyBindings(&resolved, groupsByEntity[account.EntityID], bindings)
		results = append(results, resolved)
	}
	return results, nil
}

// FirstName is the in-game first name for one entity, or "" if Sentinel has
// none or cannot be reached.
func FirstName(ctx context.Context, entityID string) string {
	return displayNames(ctx, []string{entityID})[entityID]
}

// displayNames maps entity IDs to the first name shown in game. Cosmetic, so a
// Sentinel failure is logged and the plugin falls back to Minecraft names
// rather than failing the resolve. Sentinel only exposes the combined name
// here, so the first word stands in for the first name.
func displayNames(ctx context.Context, entityIDs []string) map[string]string {
	names := make(map[string]string, len(entityIDs))
	summaries, err := sentinel.ResolveIdentities(ctx, config.SentinelSAToken, entityIDs)
	if err != nil {
		logger.SugarLogger.Warnf("failed to resolve display names: %v", err)
		return names
	}
	for _, summary := range summaries {
		if fields := strings.Fields(summary.Name); len(fields) > 0 {
			names[summary.ID] = fields[0]
		}
	}
	return names
}

// applyBindings intersects the entity's Sentinel groups with the configured
// bindings and collects the union of what they grant. Matching is on group
// ID, never name — names are editable in Sentinel and a rename must not
// silently revoke access.
func applyBindings(resolved *ResolvedPermissions, groups []sentinel.Group, bindings []model.GroupPermissionBinding) {
	bindingsByGroupID := make(map[string]model.GroupPermissionBinding, len(bindings))
	for _, binding := range bindings {
		bindingsByGroupID[binding.GroupID] = binding
	}

	groupNames := newStringSet()
	luckPermsGroups := newStringSet()
	permissions := newStringSet()

	for _, group := range groups {
		binding, ok := bindingsByGroupID[group.ID]
		if !ok {
			continue
		}
		// Only bound groups are reported. The sweep never learns about a
		// player's unbound groups, so listing them here would make the two
		// resolve paths disagree for no benefit — and "which groups earned
		// this" is the useful answer anyway.
		groupNames.add(group.Name)
		luckPermsGroups.add(binding.LuckPermsGroup)
		for _, node := range binding.Permissions {
			permissions.add(node)
		}
	}

	resolved.SentinelGroups = groupNames.sorted()
	resolved.LuckPermsGroups = luckPermsGroups.sorted()
	resolved.Permissions = permissions.sorted()
}

type stringSet map[string]struct{}

func newStringSet() stringSet {
	return make(stringSet)
}

func (s stringSet) add(value string) {
	if value != "" {
		s[value] = struct{}{}
	}
}

func (s stringSet) sorted() []string {
	values := make([]string, 0, len(s))
	for value := range s {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
