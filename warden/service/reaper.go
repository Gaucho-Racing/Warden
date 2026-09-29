package service

import (
	"time"

	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

const linkTokenReapInterval = 10 * time.Minute

// StartLinkTokenReaper drops expired, unconsumed link tokens on an interval.
// Expiry is already enforced on read, so this is housekeeping rather than a
// security control — it keeps the table from growing without bound on a
// server where players spam the join flow.
func StartLinkTokenReaper() {
	go func() {
		ticker := time.NewTicker(linkTokenReapInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := PurgeExpiredLinkTokens(); err != nil {
				logger.SugarLogger.Errorf("failed to purge expired link tokens: %v", err)
			}
		}
	}()
}
