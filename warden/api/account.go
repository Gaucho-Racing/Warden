package api

import (
	"errors"
	"net/http"

	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

// ListAccounts returns every linked account to any signed-in member, so the
// roster of who is who in game is visible to the whole team. Unlinking stays
// limited to admins and the account's owner (DeleteAccount).
func ListAccounts(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	accounts, err := service.ListAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	accounts = service.HydrateIdentities(c.Request.Context(), accounts)
	c.JSON(http.StatusOK, service.HydrateStats(accounts))
}

func GetMyAccount(c *gin.Context) {
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
	c.JSON(http.StatusOK, account)
}

func GetAccount(c *gin.Context) {
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
	// Readable by the owner or an admin — same rule as the list, scoped to
	// the single row.
	Require(c, Any(RequestUserIsMinecraftAdmin(c), RequestTokenHasEntityID(c, account.EntityID)))
	c.JSON(http.StatusOK, account)
}

// DeleteAccount unlinks a Minecraft account. Owners can unlink themselves;
// admins can unlink anyone. Unlinking strips every Warden-managed permission
// on the next reconcile, which is the intended effect when someone leaves.
func DeleteAccount(c *gin.Context) {
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

	if err := service.DeleteAccount(uuid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionAccountUnlinked,
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
	c.Status(http.StatusNoContent)
}

func ListAuditLogs(c *gin.Context) {
	Require(c, RequestUserIsMinecraftAdmin(c))
	logs, err := service.ListAuditLogs(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
