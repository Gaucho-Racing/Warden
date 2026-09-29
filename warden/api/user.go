package api

import (
	"net/http"

	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gin-gonic/gin"
)

// GetCurrentUser proxies the caller's own Sentinel profile using their own
// access token, so Warden never widens what they can see.
func GetCurrentUser(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	user, err := sentinel.GetCurrentUser(c.Request.Context(), GetRequestToken(c), GetRequestTokenUserID(c))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// ListSentinelGroups backs the binding editor's group picker.
func ListSentinelGroups(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	groups, err := sentinel.GetGroups(c.Request.Context(), GetRequestToken(c))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}
