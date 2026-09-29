package api

import (
	"errors"
	"net/http"

	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

// linkTokenResponse is the confirm page's view of a pending link. It
// deliberately exposes only the Minecraft side — the page's job is to let a
// signed-in user confirm "yes, that's my account", so it needs the username
// and skin and nothing else.
type linkTokenResponse struct {
	Token     string `json:"token"`
	UUID      string `json:"uuid"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	ExpiresAt string `json:"expires_at"`
}

func GetLinkToken(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	token, err := service.GetLinkToken(c.Param("token"))
	if err != nil {
		respondLinkTokenError(c, err)
		return
	}
	if !token.IsUsable(nowUTC()) {
		respondLinkTokenError(c, linkTokenStateError(token))
		return
	}
	c.JSON(http.StatusOK, linkTokenResponse{
		Token:     token.ID,
		UUID:      token.UUID,
		Username:  token.Username,
		AvatarURL: model.AvatarURL(token.UUID),
		ExpiresAt: token.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// ConsumeLinkToken is the moment the two authenticated halves are joined:
// the UUID Mojang vouched for at join, and the entity Sentinel vouched for
// at login. The entity comes from the bearer, never from the request body,
// so a token holder can only ever link to themselves.
func ConsumeLinkToken(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	account, err := service.ConsumeLinkToken(c.Param("token"), GetRequestTokenEntityID(c))
	if err != nil {
		respondLinkTokenError(c, err)
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionAccountLinked,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		MinecraftUUID:   account.UUID,
		MinecraftName:   account.Username,
		TargetID:        account.EntityID,
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.JSON(http.StatusOK, account)
}

func linkTokenStateError(token model.LinkToken) error {
	if token.IsConsumed() {
		return service.ErrLinkTokenConsumed
	}
	return service.ErrLinkTokenExpired
}

func respondLinkTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLinkTokenNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrLinkTokenExpired),
		errors.Is(err, service.ErrLinkTokenConsumed):
		c.JSON(http.StatusGone, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUUIDAlreadyLinked),
		errors.Is(err, service.ErrEntityAlreadyLinked):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
