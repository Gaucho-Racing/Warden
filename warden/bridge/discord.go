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
	"github.com/gaucho-racing/warden/warden/service"
)

const (
	webhookName      = "Warden"
	maxGameMessage   = 256
	outboundCapacity = 200
	// Short enough that a Sentinel outage delays each message briefly before
	// falling back to the plain name, rather than stalling the queue.
	resolveTimeout = 2 * time.Second

	colorJoin = 0x57F287
	colorQuit = 0xED4245
	// Discord treats 0x000000 as "no colour" and draws its default gray bar.
	colorDeath       = 0x010101
	colorAdvancement = 0xFEE75C
)

var customEmoji = regexp.MustCompile(`<a?:(\w+):\d+>`)

// Discord rejects webhook usernames containing these, case-insensitively.
var reservedUsername = regexp.MustCompile(`(?i)discord|clyde`)

// Bridge owns the plugin's WebSocket and, when Discord is configured, the
// relay between that socket and one Discord channel. Game chat goes through
// a channel webhook so each message carries the player's name and head;
// joins, deaths and other notices are posted by the bot itself. Discord to
// game goes back over the plugin WebSocket.
//
// The plugin socket is not conditional on Discord. It is also how Warden
// commands the game server — backups, in particular — so it must come up
// whether or not a bot token is set.
type Bridge struct {
	// session is nil when Discord is unconfigured or failed to connect.
	// Everything Discord-side checks it first; the plugin socket does not.
	session   *discordgo.Session
	botID     string
	channelID string
	hub       *Hub
	outbound  chan outboundMessage
	presence  presence

	webhookMu sync.Mutex
	webhook   *discordgo.Webhook
}

var current *Bridge

// Current is nil only before Start has run.
func Current() *Bridge {
	return current
}

// DiscordConnected reports whether game events are reaching Discord.
func (b *Bridge) DiscordConnected() bool {
	return b.session != nil
}

// PluginConnected reports whether a game server is holding the socket open.
// Commands are fire-and-forget broadcasts, so callers that need the game
// server to actually act check this first rather than waiting for a reply
// that will never come.
func (b *Bridge) PluginConnected() bool {
	return b.hub.Count() > 0
}

// Start brings up the plugin socket, then connects Discord if it is
// configured. A Discord failure is logged and leaves the relay off: the game
// server must keep working through a Discord outage, and a missing bot token
// in development must not take the plugin socket with it.
func Start() {
	b := &Bridge{
		hub:      NewHub(),
		outbound: make(chan outboundMessage, outboundCapacity),
	}
	current = b
	service.SetGameLink(b)

	if !config.DiscordBridgeEnabled() {
		logger.SugarLogger.Infof("bridge: DISCORD_TOKEN or DISCORD_CHANNEL_ID not set, Discord relay disabled")
		return
	}
	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		logger.SugarLogger.Errorf("bridge: create Discord session: %v", err)
		return
	}
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent
	session.AddHandler(b.onDiscordMessage)
	session.AddHandler(b.onReady)
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
	b.channelID = config.DiscordChannelID
	b.session = session
	go b.deliver()
	go b.runPresence()
	logger.SugarLogger.Infof("bridge: connected to Discord, relaying channel %s", b.channelID)
}

// Announce posts an operational notice to Discord and into game chat. Both
// are best effort: neither a disconnected game server nor a Discord outage
// is a reason to fail whatever the notice was about.
//
// Notices are written for Discord and rewritten for the game, because the
// two render almost nothing in common.
func (b *Bridge) Announce(text string) {
	b.hub.Broadcast(Announcement{Type: MessageAnnouncement, Text: gameText(text)})
	b.queue(outboundMessage{notice: serverMessage(text)})
}

// gameText rewrites a notice written for Discord into something Minecraft
// chat can show: bold markers appear literally there, and the default font
// has no glyph for anything outside Latin-1, drawing a missing-character box
// instead. Collapsing whitespace afterwards closes the gap an emoji leaves.
//
// Punctuation is transliterated rather than dropped. Deleting an em dash
// turns "saving — expect lag" into "saving expect lag", which reads as a
// typo; every other non-Latin-1 rune is decoration and can simply go.
var gameTextReplacer = strings.NewReplacer(
	"**", "",
	"—", "-",
	"–", "-",
	"…", "...",
	"‘", "'", "’", "'",
	"“", `"`, "”", `"`,
)

func gameText(text string) string {
	var stripped strings.Builder
	stripped.Grow(len(text))
	for _, r := range gameTextReplacer.Replace(text) {
		if r <= 0xFF {
			stripped.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(stripped.String()), " ")
}

// StartBackup hands the game server a presigned URL and tells it to archive
// itself. It reports only whether the command was dispatched; the outcome
// comes back later over the plugin's HTTP API.
func (b *Bridge) StartBackup(jobID string, uploadURL string, method string, contentType string, fileName string) error {
	if !b.PluginConnected() {
		return service.ErrGameServerOffline
	}
	b.hub.Broadcast(BackupCommand{
		Type:        MessageBackupStart,
		JobID:       jobID,
		UploadURL:   uploadURL,
		Method:      method,
		ContentType: contentType,
		FileName:    fileName,
	})
	return nil
}

func (b *Bridge) queue(message outboundMessage) {
	if b.session == nil {
		return
	}
	select {
	case b.outbound <- message:
	default:
		logger.SugarLogger.Warnf("bridge: Discord queue full, dropping message")
	}
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
	name, username := discordAuthor(ctx, m.Author.ID, discordDisplayName(m))
	b.hub.Broadcast(DiscordMessage{Type: MessageDiscord, Name: name, Username: username, Text: text})
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
	// Server state is recorded whether or not Discord is listening, so it
	// is handled before the relay bails out.
	if event.Type == EventServer && event.State == "stopping" {
		b.markStopping()
	}
	if b.session == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()

	var message outboundMessage
	switch event.Type {
	case EventChat:
		if strings.TrimSpace(event.Text) == "" {
			return
		}
		message.chat = &discordgo.WebhookParams{
			Username:        webhookUsername(playerName(ctx, event.UUID, event.Username)),
			AvatarURL:       model.AvatarURL(event.UUID),
			Content:         event.Text,
			AllowedMentions: noMentions(),
		}
	case EventJoin:
		message.notice = b.playerNotice(ctx, event, "joined the server", colorJoin)
	case EventQuit:
		message.notice = b.playerNotice(ctx, event, "left the server", colorQuit)
	case EventDeath:
		message.notice = notice(event.Text+" 💀", event.UUID, colorDeath)
	case EventAdvancement:
		message.notice = b.playerNotice(ctx, event, "has made the advancement "+event.Text+"!", colorAdvancement)
	case EventServer:
		switch event.State {
		case "started":
			message.notice = serverMessage("✅ **Server has started**")
		case "stopping":
			message.notice = serverMessage("🛑 **Server has stopped**")
		}
	}
	if message.chat == nil && message.notice == nil {
		return
	}
	b.queue(message)
}

// outboundMessage is exactly one of a chat line, posted through the webhook
// as the player, or a notice embed, posted by the bot. They share one queue
// so Discord shows them in the order they happened.
type outboundMessage struct {
	chat   *discordgo.WebhookParams
	notice *discordgo.MessageSend
}

// noMentions stops anything the game sends from pinging anyone.
func noMentions() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
}

func (b *Bridge) playerNotice(ctx context.Context, event Event, action string, color int) *discordgo.MessageSend {
	return notice(playerName(ctx, event.UUID, event.Username)+" "+action, event.UUID, color)
}

// notice is a one-line embed: the player's head and the event on the author
// line. That line is plain text, so markdown in player-controlled text (a
// named item in a death message, say) is shown literally, never rendered.
func notice(title string, uuid string, color int) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{{
			Author: &discordgo.MessageEmbedAuthor{Name: truncate(title, 256), IconURL: model.AvatarURL(uuid)},
			Color:  color,
		}},
		AllowedMentions: noMentions(),
	}
}

// serverMessage is a plain bot message rather than an embed, so server
// lifecycle stands apart from player events.
func serverMessage(content string) *discordgo.MessageSend {
	return &discordgo.MessageSend{Content: content, AllowedMentions: noMentions()}
}

// deliver sends one message at a time so Discord shows them in the order
// they happened. discordgo waits out rate limits on each call.
func (b *Bridge) deliver() {
	for message := range b.outbound {
		var err error
		if message.chat != nil {
			err = b.execute(message.chat)
		} else {
			_, err = b.session.ChannelMessageSendComplex(b.channelID, message.notice)
		}
		if err != nil {
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
