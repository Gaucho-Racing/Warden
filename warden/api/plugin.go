package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

type issueLinkTokenRequest struct {
	UUID     string `json:"uuid" binding:"required"`
	Username string `json:"username" binding:"required"`
}

type issueLinkTokenResponse struct {
	Token     string `json:"token"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// IssueLinkToken mints a pending link for a player the game server has
// already authenticated against Mojang. Warden returns the full URL rather
// than just the token so the portal can move hosts without reshipping a jar.
func IssueLinkToken(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	var req issueLinkTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uuid, username, err := normalizePlayer(req.UUID, req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := service.CreateLinkToken(uuid, username)
	if errors.Is(err, service.ErrUUIDAlreadyLinked) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:        model.AuditActionLinkTokenIssued,
		MinecraftUUID: uuid,
		MinecraftName: username,
		TargetID:      token.ID,
		RequestMethod: c.Request.Method,
		RequestPath:   c.Request.URL.Path,
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
	})
	c.JSON(http.StatusCreated, issueLinkTokenResponse{
		Token:     token.ID,
		URL:       service.LinkURL(token),
		ExpiresAt: token.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// GetPlayerPermissions returns the complete desired permission state for a
// UUID. An unlinked player is a 200 with linked=false and empty sets, not an
// error — "this player gets nothing" is a valid answer the plugin must be
// able to apply.
func GetPlayerPermissions(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	uuid, err := service.NormalizeUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resolved, err := service.ResolveForUUID(c.Request.Context(), uuid, c.Query("username"))
	if err != nil {
		// Sentinel being unreachable is the plugin's cue to fall back to its
		// own cached state, so this is a gateway error rather than a 500.
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resolved)
}

// MarkPlayerSeen refreshes the cached username and last-seen timestamp.
// Split out from the permission read so the read stays side-effect free and
// can be retried freely.
func MarkPlayerSeen(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	uuid, err := service.NormalizeUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	username := c.Query("username")
	if username != "" {
		if username, err = service.ValidateUsername(username); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	service.TouchAccount(uuid, username)
	c.Status(http.StatusNoContent)
}

// SyncAllPlayers is the reconcile sweep's endpoint: the full desired state
// of the permission system in one response. The plugin diffs it against
// what LuckPerms currently holds and corrects the difference, which is how
// a revocation lands without waiting for the player to rejoin.
//
// Apply groups before players. LuckPerms will not create a group on demand,
// and adding somebody to one that does not exist stores an inheritance node
// that resolves to nothing — no error anywhere, just a player missing every
// permission the portal says they have.
//
// Both lists are exhaustive, so the plugin can garbage collect too: a group
// carrying managed_group_prefix but absent from groups belongs to a deleted
// binding, and a managed group absent from a player's entry was revoked.
func SyncAllPlayers(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	groups, err := service.ManagedGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resolved, err := service.ResolveAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"managed_group_prefix": service.ManagedGroupPrefix,
		"groups":               groups,
		"players":              resolved,
	})
}

func normalizePlayer(rawUUID string, rawUsername string) (string, string, error) {
	uuid, err := service.NormalizeUUID(rawUUID)
	if err != nil {
		return "", "", err
	}
	username, err := service.ValidateUsername(rawUsername)
	if err != nil {
		return "", "", err
	}
	return uuid, username, nil
}

type serverStatusReport struct {
	Online        int      `json:"online"`
	Players       []string `json:"players"`
	MaxPlayers    int      `json:"max_players"`
	UniquePlayers int      `json:"unique_players"`
	TPS           float64  `json:"tps"`
	MSPT          float64  `json:"mspt"`
	// Epoch milliseconds, as the JVM reports its own start time.
	StartedAt int64 `json:"started_at"`
	// Sent once from onDisable in place of a sample.
	Stopping bool `json:"stopping"`
}

// ReportServerStatus ingests the plugin's periodic server health sample. It
// is recorded whether or not the Discord bridge is running.
func ReportServerStatus(c *gin.Context) {
	Require(c, RequestIsPlugin(c))
	var report serverStatusReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if report.Stopping {
		service.MarkServerStopping()
		c.Status(http.StatusNoContent)
		return
	}
	players := make([]string, 0, len(report.Players))
	for _, raw := range report.Players {
		uuid, err := service.NormalizeUUID(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		players = append(players, uuid)
	}
	status := model.ServerStatus{
		RecordedAt:    time.Now(),
		Online:        report.Online,
		OnlinePlayers: players,
		MaxPlayers:    report.MaxPlayers,
		UniquePlayers: report.UniquePlayers,
		TPS:           report.TPS,
		MSPT:          report.MSPT,
		StartedAt:     time.UnixMilli(report.StartedAt),
	}
	if err := service.RecordServerStatus(status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
