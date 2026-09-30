package service

import (
	"context"
	"errors"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"gorm.io/gorm"
)

var ErrAccountNotFound = errors.New("no minecraft account is linked")

func GetAccountByUUID(uuid string) (model.MinecraftAccount, error) {
	var account model.MinecraftAccount
	if err := database.DB.Where("uuid = ?", uuid).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.MinecraftAccount{}, ErrAccountNotFound
		}
		return model.MinecraftAccount{}, err
	}
	return account, nil
}

func GetAccountByEntityID(entityID string) (model.MinecraftAccount, error) {
	var account model.MinecraftAccount
	if err := database.DB.Where("entity_id = ?", entityID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.MinecraftAccount{}, ErrAccountNotFound
		}
		return model.MinecraftAccount{}, err
	}
	return account, nil
}

func ListAccounts() ([]model.MinecraftAccount, error) {
	accounts := []model.MinecraftAccount{}
	if err := database.DB.Order("created_at desc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func CreateAccount(account model.MinecraftAccount) (model.MinecraftAccount, error) {
	if err := database.DB.Create(&account).Error; err != nil {
		return model.MinecraftAccount{}, err
	}
	return account, nil
}

func DeleteAccount(uuid string) error {
	return database.DB.Where("uuid = ?", uuid).Delete(&model.MinecraftAccount{}).Error
}

// TouchAccount records that a UUID was seen on the server and refreshes the
// cached username. Minecraft usernames change; the UUID does not, so the
// stored name is a display cache that we keep current on every join.
func TouchAccount(uuid string, username string) {
	updates := map[string]any{"last_seen_at": time.Now()}
	if username != "" {
		updates["username"] = username
	}
	if err := database.DB.Model(&model.MinecraftAccount{}).Where("uuid = ?", uuid).Updates(updates).Error; err != nil {
		logger.SugarLogger.Errorf("failed to touch account %s: %v", uuid, err)
	}
}

// HydrateIdentities fills in the Sentinel-side display fields for a batch of
// accounts in one call. Best-effort: if Sentinel is unreachable the accounts
// come back with EntityID only rather than failing the whole response.
func HydrateIdentities(ctx context.Context, accounts []model.MinecraftAccount) []model.MinecraftAccount {
	if len(accounts) == 0 {
		return accounts
	}
	entityIDs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if account.EntityID != "" {
			entityIDs = append(entityIDs, account.EntityID)
		}
	}
	summaries, err := sentinel.ResolveIdentities(ctx, config.SentinelSAToken, entityIDs)
	if err != nil {
		logger.SugarLogger.Warnf("failed to resolve identities: %v", err)
		return accounts
	}
	byEntityID := make(map[string]sentinel.IdentitySummary, len(summaries))
	for _, summary := range summaries {
		byEntityID[summary.ID] = summary
	}
	for i := range accounts {
		summary, ok := byEntityID[accounts[i].EntityID]
		if !ok {
			continue
		}
		accounts[i].Identity = &model.Identity{
			EntityID:  summary.ID,
			Name:      summary.Name,
			Username:  summary.Username,
			AvatarURL: summary.AvatarURL,
		}
	}
	return accounts
}

// HydrateStats attaches playtime and session counts for the roster in one
// query. A failure is logged and leaves accounts without stats rather than
// failing the listing.
func HydrateStats(accounts []model.MinecraftAccount) []model.MinecraftAccount {
	if len(accounts) == 0 {
		return accounts
	}
	uuids := make([]string, len(accounts))
	for i, account := range accounts {
		uuids[i] = account.UUID
	}
	var stats []model.PlayerStats
	if err := database.DB.Select("uuid", "playtime_minutes", "sessions").Where("uuid IN ?", uuids).Find(&stats).Error; err != nil {
		logger.SugarLogger.Warnf("failed to load roster stats: %v", err)
		return accounts
	}
	byUUID := make(map[string]model.PlayerStats, len(stats))
	for _, row := range stats {
		byUUID[row.UUID] = row
	}
	for i := range accounts {
		if row, ok := byUUID[accounts[i].UUID]; ok {
			accounts[i].Stats = &model.AccountStats{PlaytimeMinutes: row.PlaytimeMinutes, Sessions: row.Sessions}
		}
	}
	return accounts
}
