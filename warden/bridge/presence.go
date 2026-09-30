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
	// The plugin reports every minute; missing a couple of reports means the
	// server is gone even if it never said "stopping" (a crash, say).
	statusStaleAfter = 150 * time.Second
)

// presence tracks what was last sent to Discord so unchanged status and topic
// are never resent.
type presence struct {
	mu        sync.Mutex
	stoppedAt time.Time
	lastState string
	lastTopic string
}

// serverState is "active" (players online), "empty" or "offline".
func (p *presence) serverState() (string, bool) {
	status := service.LatestServerStatus()
	p.mu.Lock()
	stoppedAt := p.stoppedAt
	p.mu.Unlock()
	switch {
	case status == nil:
		return "offline", false
	case status.RecordedAt.Before(stoppedAt) || time.Since(status.RecordedAt) > statusStaleAfter:
		return "offline", true
	case status.Online == 0:
		return "empty", true
	default:
		return "active", true
	}
}

func (b *Bridge) markStopping() {
	b.presence.mu.Lock()
	b.presence.stoppedAt = time.Now()
	b.presence.mu.Unlock()
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
	state, _ := b.presence.serverState()
	b.presence.mu.Lock()
	unchanged := state == b.presence.lastState
	b.presence.mu.Unlock()
	if unchanged {
		return
	}
	dot := map[string]string{"active": "online", "empty": "idle", "offline": "dnd"}[state]
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
	state, known := b.presence.serverState()
	if !known {
		return
	}
	status := service.LatestServerStatus()
	topic := fmt.Sprintf("Server offline | %d unique players ever joined", status.UniquePlayers)
	if state != "offline" {
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
