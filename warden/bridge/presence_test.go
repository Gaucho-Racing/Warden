package bridge

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

// Regression: READY arrives on a goroutine discordgo starts inside Open(),
// which can run before Start() has finished assigning b.session. discordgo
// locks the session before checking anything, so an unset one panics rather
// than returning an error, and that crash-looped the pod.
func TestOnReadySurvivesAnUnsetSession(t *testing.T) {
	b := &Bridge{hub: NewHub(), outbound: make(chan outboundMessage, 1)}
	b.onReady(nil, &discordgo.Ready{})
}
