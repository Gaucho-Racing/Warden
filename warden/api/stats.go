package api

import (
	"errors"
	"net/http"

	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

// GetMyStats returns the caller's own Minecraft stat line. A 404 means they
// have no linked account, which the portal renders as the onboarding state
// rather than an error.
func GetMyStats(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	account, err := service.GetAccountByEntityID(GetRequestTokenEntityID(c))
	if errors.Is(err, service.ErrAccountNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	stats, err := service.PlayerStatsForUUID(account.UUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPlayerStats returns a stat line by UUID. Same trust level as reading the
// account itself: any signed-in member.
func GetPlayerStats(c *gin.Context) {
	uuid, err := service.NormalizeUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = service.GetAccountByUUID(uuid)
	if errors.Is(err, service.ErrAccountNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	Require(c, RequestTokenExists(c))

	stats, err := service.PlayerStatsForUUID(uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ReportPlayerStats ingests a stat report from the plugin.
//
// The plugin is the only thing that can read these — they live in the
// server's own statistics store — so this is the sole write path. Reports
// are whole-state overwrites rather than increments: Minecraft keeps
// running totals, so a replayed or duplicated report is harmless.
func ReportPlayerStats(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	uuid, err := service.NormalizeUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var stats model.PlayerStats
	if err := c.ShouldBindJSON(&stats); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stats.UUID = uuid

	// Only report on accounts Warden knows about. An unlinked player's
	// stats have nobody to attribute them to.
	account, err := service.GetAccountByUUID(uuid)
	if errors.Is(err, service.ErrAccountNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stats.Username == "" {
		stats.Username = account.Username
	}

	if err := service.RecordPlayerStats(stats); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
