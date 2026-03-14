package discord

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/Ender-events/koncord/internal/chat"
)

// Bot implements chat.Platform for Discord using application commands.
type Bot struct {
	session    *discordgo.Session
	guildID    string
	handler    chat.CommandHandler
	logger     *slog.Logger
	commandIDs []string // registered command IDs for cleanup
}

// Ensure interface compliance.
var _ chat.Platform = (*Bot)(nil)

// New creates a new Discord bot. The handler is invoked for every slash command.
func New(token, guildID string, handler chat.CommandHandler, logger *slog.Logger) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	return &Bot{
		session: session,
		guildID: guildID,
		handler: handler,
		logger:  logger,
	}, nil
}

// Start connects to Discord and registers application commands.
func (b *Bot) Start(_ context.Context) error {
	b.session.AddHandler(b.onInteractionCreate)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}

	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}

	b.logger.Info("discord bot started", "user", b.session.State.User.Username)
	return nil
}

// Stop unregisters commands and closes the session.
func (b *Bot) Stop() error {
	for _, id := range b.commandIDs {
		if err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.guildID, id); err != nil {
			b.logger.Warn("failed to delete command", "id", id, "err", err)
		}
	}
	return b.session.Close()
}

// SendMessage sends a plain-text message to a channel.
func (b *Bot) SendMessage(_ context.Context, channelID, content string) error {
	// Discord has a 2000 char limit per message; chunk if needed.
	for len(content) > 0 {
		chunk := content
		if len(chunk) > 1990 {
			chunk = chunk[:1990]
		}
		if _, err := b.session.ChannelMessageSend(channelID, chunk); err != nil {
			return err
		}
		content = content[len(chunk):]
	}
	return nil
}

// ---------- Application command definitions ----------

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "koncord",
			Description: "Manage containers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "list",
					Description: "List all managed containers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "status",
					Description: "Show container status",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "container",
							Description: "Container name (optional if channel has a bound container)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "restart",
					Description: "Restart a container",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "container",
							Description: "Container name (optional if channel has a bound container)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "logs",
					Description: "Forward container logs to this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "container",
							Description: "Container name (optional if channel has a bound container)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "admin-register",
					Description: "Promote a user to admin",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "User to promote",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "user-register",
					Description: "Register a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "User to register",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "bind",
					Description: "Bind a container to this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "container",
							Description: "Container name to bind",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
	}

	for _, cmd := range commands {
		registered, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.guildID, cmd)
		if err != nil {
			return fmt.Errorf("create command %s: %w", cmd.Name, err)
		}
		b.commandIDs = append(b.commandIDs, registered.ID)
		b.logger.Debug("registered command", "name", cmd.Name, "id", registered.ID)
	}

	return nil
}

// ---------- Interaction handler ----------

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	if data.Name != "koncord" {
		return
	}

	if len(data.Options) == 0 {
		return
	}

	subCmd := data.Options[0]
	args := make(map[string]string)
	for _, opt := range subCmd.Options {
		switch opt.Type {
		case discordgo.ApplicationCommandOptionUser:
			args[opt.Name] = opt.UserValue(s).ID
		case discordgo.ApplicationCommandOptionString:
			args[opt.Name] = opt.StringValue()
		default:
			args[opt.Name] = fmt.Sprintf("%v", opt.Value)
		}
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	cmd := chat.CommandContext{
		UserID:    userID,
		ChannelID: i.ChannelID,
		GuildID:   i.GuildID,
		Command:   subCmd.Name,
		Args:      args,
		Respond: func(content string) error {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}

	if err := b.handler(context.Background(), cmd); err != nil {
		b.logger.Error("command failed", "command", subCmd.Name, "err", err)
		// Try to respond with error; if already responded, edit.
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error: %v", err),
			},
		})
	}
}
