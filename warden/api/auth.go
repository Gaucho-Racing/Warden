package api

import (
	"fmt"
	"net/http"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gin-gonic/gin"
)

func LoginWithSentinel(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	token, err := sentinel.ExchangeAuthorizationCode(code, callbackURL(c))
	if err != nil {
		logger.SugarLogger.Errorln("Failed to exchange Sentinel authorization code: " + err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}

func GetSession(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	c.JSON(http.StatusOK, gin.H{
		"entity_id":          GetRequestTokenEntityID(c),
		"user_id":            GetRequestTokenUserID(c),
		"scope":              GetRequestTokenScopes(c),
		"groups":             GetRequestTokenGroupNames(c),
		"is_minecraft_admin": RequestUserIsMinecraftAdmin(c),
	})
}

func RefreshSession(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}
	token, err := sentinel.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, token)
}

func Logout(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func callbackURL(c *gin.Context) string {
	if config.SentinelRedirectURI != "" {
		return config.SentinelRedirectURI
	}
	return requestBaseURL(c) + "/auth/login"
}

func requestBaseURL(c *gin.Context) string {
	proto := c.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s", proto, host)
}
