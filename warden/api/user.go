package api

import (
	"net/http"

	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gaucho-racing/warden/warden/service"
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

// ListBindableGroups backs the binding editor's group picker. It returns
// only the groups linked to the Warden application in Sentinel, resolved
// with Warden's own service account rather than the caller's token — the
// answer is a property of the app, not of who is asking.
func ListBindableGroups(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	groups, err := service.BindableGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}
