package bridge

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

const (
	webhookName      = "Warden"
	maxGameMessage   = 256
	outboundCapacity = 200
	resolveTimeout   = 5 * time.Second

	colorJoin        = 0x57F287
	colorQuit        = 0xED4245
	colorDeath       = 0x99AAB5
	colorAdvancement = 0xFEE75C
)

var customEmoji = regexp.MustCompile(`<a?:(\w+):\d+>`)

// Discord rejects webhook usernames containing these, case-insensitively.
var reservedUsername = regexp.MustCompile(`(?i)discord|clyde`)

// Bridge relays chat between the game and one Discord channel. Game to
// Discord goes through a channel webhook so each message carries the
// player's name and head; Discord to game goes over the plugin WebSocket.
type Bridge struct {
	session   *discordgo.Session
	botID     string
	channelID string
	hub       *Hub
	names     *names
	outbound  chan *discordgo.WebhookParams

	webhookMu sync.Mutex
	webhook   *discordgo.Webhook
}

var current *Bridge

// Current is nil when the bridge is disabled or failed to start.
func Current() *Bridge {
	return current
}

// Start connects the bot. A failure is logged and leaves the bridge off, so
// a Discord outage never stops Warden itself from serving the game.
func Start() {
	if !config.DiscordBridgeEnabled() {
		logger.SugarLogger.Infof("bridge: DISCORD_TOKEN or DISCORD_CHANNEL_ID not set, Discord bridge disabled")
		return
	}
	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		logger.SugarLogger.Errorf("bridge: create Discord session: %v", err)
		return
	}
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	b := &Bridge{
		session:   session,
		channelID: config.DiscordChannelID,
		hub:       NewHub(),
		names:     newNames(),
		outbound:  make(chan *discordgo.WebhookParams, outboundCapacity),
	}
	session.AddHandler(b.onDiscordMessage)
	if err := session.Open(); err != nil {
		logger.SugarLogger.Errorf("bridge: open Discord gateway: %v", err)
		return
	}
	me, err := session.User("@me")
	if err != nil {
		logger.SugarLogger.Errorf("bridge: look up bot user: %v", err)
		session.Close()
		return
	}
	b.botID = me.ID
	go b.deliver()
	current = b
	logger.SugarLogger.Infof("bridge: connected to Discord, relaying channel %s", b.channelID)
}

// ServePlugin handles the plugin's WebSocket.
func (b *Bridge) ServePlugin(w http.ResponseWriter, r *http.Request) {
	b.hub.Serve(w, r, b.onGameEvent)
}

func (b *Bridge) onDiscordMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Our own webhook posts come back as messages; relaying them would echo
	// every game message straight back into the game.
	if m.ChannelID != b.channelID || m.Author == nil || m.Author.Bot || m.WebhookID != "" {
		return
	}
	text := b.renderForGame(s, m.Message)
	if text == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()
	name := b.names.discordAuthor(ctx, m.Author.ID, discordDisplayName(m))
	b.hub.Broadcast(DiscordMessage{Type: MessageDiscord, Name: name, Text: text})
}

// renderForGame flattens a Discord message to one plain line. Mentions become
// readable names, custom emoji their :name:, and attachments a placeholder,
// since none of those render in Minecraft chat.
func (b *Bridge) renderForGame(s *discordgo.Session, m *discordgo.Message) string {
	content, err := m.ContentWithMoreMentionsReplaced(s)
	if err != nil {
		content = m.ContentWithMentionsReplaced()
	}
	content = customEmoji.ReplaceAllString(content, ":$1:")
	parts := strings.Fields(content)
	for range m.Attachments {
		parts = append(parts, "[attachment]")
	}
	for range m.StickerItems {
		parts = append(parts, "[sticker]")
	}
	return truncate(strings.Join(parts, " "), maxGameMessage)
}

func discordDisplayName(m *discordgo.MessageCreate) string {
	if m.Member != nil && m.Member.Nick != "" {
		return m.Member.Nick
	}
	if m.Author.GlobalName != "" {
		return m.Author.GlobalName
	}
	return m.Author.Username
}

func (b *Bridge) onGameEvent(event Event) {
	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()

	var params *discordgo.WebhookParams
	switch event.Type {
	case EventChat:
		if strings.TrimSpace(event.Text) == "" {
			return
		}
		params = &discordgo.WebhookParams{
			Username:  webhookUsername(b.names.player(ctx, event.UUID)),
			AvatarURL: model.AvatarURL(event.UUID),
			Content:   event.Text,
		}
	case EventJoin:
		params = b.playerEmbed(ctx, event.UUID, "joined the server", colorJoin)
	case EventQuit:
		params = b.playerEmbed(ctx, event.UUID, "left the server", colorQuit)
	case EventDeath:
		params = embed(event.Text, model.AvatarURL(event.UUID), colorDeath)
	case EventAdvancement:
		name := b.names.player(ctx, event.UUID)
		params = embed(name+" has made the advancement "+event.Text, model.AvatarURL(event.UUID), colorAdvancement)
	case EventServer:
		switch event.State {
		case "started":
			params = embed("Server started", "", colorJoin)
		case "stopping":
			params = embed("Server stopping", "", colorQuit)
		}
	}
	if params == nil {
		return
	}
	// Nothing the game sends may ping anyone.
	params.AllowedMentions = &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
	select {
	case b.outbound <- params:
	default:
		logger.SugarLogger.Warnf("bridge: Discord queue full, dropping %s event", event.Type)
	}
}

func (b *Bridge) playerEmbed(ctx context.Context, uuid string, action string, color int) *discordgo.WebhookParams {
	return embed(b.names.player(ctx, uuid)+" "+action, model.AvatarURL(uuid), color)
}

func embed(title string, iconURL string, color int) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Author: &discordgo.MessageEmbedAuthor{Name: truncate(title, 256), IconURL: iconURL},
			Color:  color,
		}},
	}
}

// deliver sends webhook messages one at a time so Discord shows them in the
// order they happened. discordgo waits out rate limits on each call.
func (b *Bridge) deliver() {
	for params := range b.outbound {
		if err := b.execute(params); err != nil {
			logger.SugarLogger.Warnf("bridge: send to Discord: %v", err)
		}
	}
}

func (b *Bridge) execute(params *discordgo.WebhookParams) error {
	webhook, err := b.channelWebhook()
	if err != nil {
		return err
	}
	_, err = b.session.WebhookExecute(webhook.ID, webhook.Token, false, params)
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownWebhook {
		// Someone deleted the webhook in Discord; make a new one and retry once.
		b.forgetWebhook()
		if webhook, err = b.channelWebhook(); err != nil {
			return err
		}
		_, err = b.session.WebhookExecute(webhook.ID, webhook.Token, false, params)
	}
	return err
}

// channelWebhook finds the webhook this bot created earlier, or creates one.
// Ownership is by application_id, so a hand-made webhook that happens to
// share the name is left alone. A bot's application ID is its user ID.
func (b *Bridge) channelWebhook() (*discordgo.Webhook, error) {
	b.webhookMu.Lock()
	defer b.webhookMu.Unlock()
	if b.webhook != nil {
		return b.webhook, nil
	}
	hooks, err := b.session.ChannelWebhooks(b.channelID)
	if err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		if hook.Name == webhookName && hook.ApplicationID == b.botID && hook.Token != "" {
			b.webhook = hook
			return hook, nil
		}
	}
	hook, err := b.session.WebhookCreate(b.channelID, webhookName, "")
	if err != nil {
		return nil, err
	}
	b.webhook = hook
	return hook, nil
}

func (b *Bridge) forgetWebhook() {
	b.webhookMu.Lock()
	defer b.webhookMu.Unlock()
	b.webhook = nil
}

// webhookUsername fits Discord's rules: 1-80 characters, without the
// reserved words, which are broken up with a zero-width space.
func webhookUsername(name string) string {
	name = reservedUsername.ReplaceAllStringFunc(name, func(word string) string {
		return word[:1] + "​" + word[1:]
	})
	return truncate(name, 80)
}

func truncate(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return string(runes[:limit-1]) + "…"
}
