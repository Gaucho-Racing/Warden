package service

import (
	"context"
	"sync"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
)

// The application id for a given client id never changes, so resolve it once
// and keep it. A failed resolve is not cached — the next call retries.
var (
	appIDOnce sync.Mutex
	appID     string
)

func wardenApplicationID(ctx context.Context) (string, error) {
	appIDOnce.Lock()
	defer appIDOnce.Unlock()
	if appID != "" {
		return appID, nil
	}
	app, err := sentinel.GetApplicationByClientID(ctx, config.SentinelSAToken, config.SentinelClientID)
	if err != nil {
		return "", err
	}
	appID = app.ID
	return appID, nil
}

// BindableGroups returns the Sentinel groups linked to the Warden
// application. Bindings may only ever reference one of these.
//
// Linking a group to the app in Sentinel is the deliberate act that makes it
// available to Minecraft; without that gate the binding editor would offer
// every group in the org, most of which have nothing to do with the game
// server. An empty result means nobody has linked any yet, which is a
// configuration state and not an error.
func BindableGroups(ctx context.Context) ([]sentinel.ApplicationGroup, error) {
	id, err := wardenApplicationID(ctx)
	if err != nil {
		return nil, err
	}
	return sentinel.GetApplicationGroups(ctx, config.SentinelSAToken, id)
}
