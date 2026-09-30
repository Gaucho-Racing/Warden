package api

import (
	"net/http"
	"time"

	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

type serverStatusResponse struct {
	State         string     `json:"state"`
	Online        int        `json:"online"`
	MaxPlayers    int        `json:"max_players"`
	UniquePlayers int        `json:"unique_players"`
	UptimeMinutes int        `json:"uptime_minutes"`
	TPS           float64    `json:"tps"`
	RecordedAt    *time.Time `json:"recorded_at,omitempty"`
}

// GetServerStatus is the portal's view of the game server, the same state
// the Discord presence and channel topic are built from. State is "active",
// "empty" or "offline"; the counts are from the latest sample, if any.
func GetServerStatus(c *gin.Context) {
	Require(c, RequestTokenExists(c))
	state, status := service.CurrentServerState()
	response := serverStatusResponse{State: state}
	if status != nil {
		response.Online = status.Online
		response.MaxPlayers = status.MaxPlayers
		response.UniquePlayers = status.UniquePlayers
		response.TPS = status.TPS
		response.RecordedAt = &status.RecordedAt
		if state != service.ServerOffline {
			response.UptimeMinutes = int(time.Since(status.StartedAt).Minutes())
		}
	}
	c.JSON(http.StatusOK, response)
}
