package chat

import "context"

// Platform abstracts a chat service (Discord, Matrix, …).
type Platform interface {
	// Start connects to the chat service and begins listening for commands.
	Start(ctx context.Context) error
	// Stop gracefully disconnects.
	Stop() error
	// SendMessage sends a text message to the given channel.
	SendMessage(ctx context.Context, channelID, content string) error
}

// CommandContext represents an incoming command from a user.
type CommandContext struct {
	UserID    string
	ChannelID string
	GuildID   string
	Command   string            // sub-command name, e.g. "list", "status"
	Args      map[string]string // named arguments
	Respond   func(content string) error
}

// CommandHandler processes a command. Implementations live in internal/bot.
type CommandHandler func(ctx context.Context, cmd CommandContext) error
