package bridge

// Event is a game event from the plugin. Which fields are set depends on
// Type: chat and death carry Text, advancement carries Text as the title,
// server carries State, and everything but server carries UUID and Username
// (the Minecraft name, used when the player has no linked account).
type Event struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid,omitempty"`
	Username string `json:"username,omitempty"`
	Text     string `json:"text,omitempty"`
	State    string `json:"state,omitempty"`
}

const (
	EventChat        = "chat"
	EventJoin        = "join"
	EventQuit        = "quit"
	EventDeath       = "death"
	EventAdvancement = "advancement"
	EventServer      = "server"
)

// DiscordMessage is a Discord post relayed into the game. Names are already
// resolved, so the plugin only renders them. Username is the author's
// Minecraft name, empty when they have no linked account.
type DiscordMessage struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Text     string `json:"text"`
}

const MessageDiscord = "discord_message"
