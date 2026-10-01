package main

import (
	"github.com/gaucho-racing/warden/warden/api"
	"github.com/gaucho-racing/warden/warden/bridge"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gaucho-racing/warden/warden/service"
)

func main() {
	logger.Init(config.IsProduction())
	defer logger.Logger.Sync()

	config.Verify()
	config.PrintStartupBanner()
	if err := sentinel.InitializeSigningKeys(); err != nil {
		logger.SugarLogger.Warnf("initialize Sentinel signing keys: %v", err)
	}
	database.Init()
	service.StartLinkTokenReaper()
	// The bridge registers itself as the game link, so it has to be up
	// before the scheduler can dispatch a backup to the game server.
	bridge.Start()
	service.StartBackupScheduler()

	api.Run()
}
