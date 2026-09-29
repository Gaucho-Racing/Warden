package api

import (
	"errors"
	"net/http"

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
// account itself: the owner or an admin.
func GetPlayerStats(c *gin.Context) {
	uuid, err := service.NormalizeUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := service.GetAccountByUUID(uuid)
	if errors.Is(err, service.ErrAccountNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	Require(c, Any(RequestUserIsMinecraftAdmin(c), RequestTokenHasEntityID(c, account.EntityID)))

	stats, err := service.PlayerStatsForUUID(uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
