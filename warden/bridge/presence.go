package bridge

import (
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/service"
)

const (
	presenceInterval = 15 * time.Second
	// Discord allows two channel edits per ten minutes, so the topic is
	// rewritten at most this often and only when its text changed.
	topicInterval = 10 * time.Minute
)

// presence tracks what was last sent to Discord so unchanged status and topic
// are never resent. Whether the server is up is decided by the service layer,
// so the web portal and Discord always agree.
type presence struct {
	mu        sync.Mutex
	lastState string
	lastTopic string
}

func (b *Bridge) markStopping() {
	service.MarkServerStopping()
	b.updatePresence()
}

// onReady forgets the last presence: a new gateway session starts with none.
func (b *Bridge) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.presence.mu.Lock()
	b.presence.lastState = ""
	b.presence.mu.Unlock()
	b.updatePresence()
}

func (b *Bridge) runPresence() {
	presenceTicker := time.NewTicker(presenceInterval)
	defer presenceTicker.Stop()
	topicTimer := time.NewTimer(time.Minute)
	defer topicTimer.Stop()
	for {
		select {
		case <-presenceTicker.C:
			b.updatePresence()
		case <-topicTimer.C:
			b.updateTopic()
			topicTimer.Reset(topicInterval)
		}
	}
}

// updatePresence shows "Playing Minecraft" throughout; the status dot carries
// the server state, since a bot only ever displays one activity.
func (b *Bridge) updatePresence() {
	state, _ := service.CurrentServerState()
	b.presence.mu.Lock()
	unchanged := state == b.presence.lastState
	b.presence.mu.Unlock()
	if unchanged {
		return
	}
	dot := map[string]string{service.ServerActive: "online", service.ServerEmpty: "idle", service.ServerOffline: "dnd"}[state]
	err := b.session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status:     dot,
		Activities: []*discordgo.Activity{{Name: "Minecraft", Type: discordgo.ActivityTypeGame}},
	})
	if err != nil {
		logger.SugarLogger.Warnf("bridge: update presence: %v", err)
		return
	}
	b.presence.mu.Lock()
	b.presence.lastState = state
	b.presence.mu.Unlock()
}

func (b *Bridge) updateTopic() {
	state, status := service.CurrentServerState()
	if status == nil {
		return
	}
	topic := fmt.Sprintf("Server offline | %d unique players ever joined", status.UniquePlayers)
	if state != service.ServerOffline {
		topic = fmt.Sprintf("%d/%d players online | %d unique players ever joined | Server online for %d minutes",
			status.Online, status.MaxPlayers, status.UniquePlayers, int(time.Since(status.StartedAt).Minutes()))
	}
	b.presence.mu.Lock()
	unchanged := topic == b.presence.lastTopic
	b.presence.mu.Unlock()
	if unchanged {
		return
	}
	if _, err := b.session.ChannelEdit(b.channelID, &discordgo.ChannelEdit{Topic: topic}); err != nil {
		logger.SugarLogger.Warnf("bridge: update channel topic: %v", err)
		return
	}
	b.presence.mu.Lock()
	b.presence.lastTopic = topic
	b.presence.mu.Unlock()
}
