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

// Announcement is an operational notice from Warden — a backup warning, say
// — rendered as a highlighted line in game chat.
//
// It is pushed over this socket rather than left to the Discord relay,
// because the relay deliberately ignores bot and webhook messages: without
// that filter every game line Warden echoed to Discord would come straight
// back into the game. So a notice Warden posts to Discord does not reach
// players, and has to be sent to both places explicitly.
type Announcement struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// BackupCommand tells the game server to archive itself and PUT the result
// to UploadURL. The URL is presigned against object storage, so the plugin
// uploads without ever holding a Depot or Sentinel credential, and the
// archive never passes through Warden.
//
// ContentType must be echoed verbatim on the PUT: it is part of the
// presignature, and object storage rejects a mismatch.
type BackupCommand struct {
	Type        string `json:"type"`
	JobID       string `json:"job_id"`
	UploadURL   string `json:"upload_url"`
	Method      string `json:"method"`
	ContentType string `json:"content_type"`
	FileName    string `json:"file_name"`
}

const (
	MessageDiscord      = "discord_message"
	MessageAnnouncement = "announcement"
	MessageBackupStart  = "backup_start"
)
