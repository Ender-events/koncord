package bot

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Ender-events/koncord/internal/auth"
	"github.com/Ender-events/koncord/internal/chat"
	"github.com/Ender-events/koncord/internal/domain"
	"github.com/Ender-events/koncord/internal/runtime"
	"github.com/Ender-events/koncord/internal/store"
)

// Bot is the central command router that ties together auth, runtime, chat,
// and persistence.
type Bot struct {
	auth     *auth.Manager
	runtime  runtime.ContainerRuntime
	platform chat.Platform
	store    *store.Store
	logger   *slog.Logger

	// log forwarding
	logMu      sync.Mutex
	logCancel  map[string]context.CancelFunc // channelID → cancel
}

// New creates a new Bot.
func New(
	authMgr *auth.Manager,
	rt runtime.ContainerRuntime,
	platform chat.Platform,
	st *store.Store,
	logger *slog.Logger,
) *Bot {
	return &Bot{
		auth:      authMgr,
		runtime:   rt,
		platform:  platform,
		store:     st,
		logger:    logger,
		logCancel: make(map[string]context.CancelFunc),
	}
}

// HandleCommand is the chat.CommandHandler entry point.
func (b *Bot) HandleCommand(ctx context.Context, cmd chat.CommandContext) error {
	// Auto-promote first user to super-admin.
	b.auth.EnsureInitialised(cmd.UserID)

	switch cmd.Command {
	case "list":
		return b.handleList(ctx, cmd)
	case "status":
		return b.handleStatus(ctx, cmd)
	case "restart":
		return b.handleRestart(ctx, cmd)
	case "logs":
		return b.handleLogs(ctx, cmd)
	case "admin-register":
		return b.handleAdminRegister(ctx, cmd)
	case "user-register":
		return b.handleUserRegister(ctx, cmd)
	case "bind":
		return b.handleBind(ctx, cmd)
	default:
		return cmd.Respond("❓ Unknown command: " + cmd.Command)
	}
}

// StopAllLogs cancels every active log-forwarding goroutine.
func (b *Bot) StopAllLogs() {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	for ch, cancel := range b.logCancel {
		cancel()
		delete(b.logCancel, ch)
	}
}

// ---------- command handlers ----------

func (b *Bot) handleList(ctx context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleUser) {
		return cmd.Respond("🚫 You are not registered. Ask an admin to register you.")
	}

	containers, err := b.runtime.ListContainers(ctx)
	if err != nil {
		return cmd.Respond(fmt.Sprintf("❌ Failed to list containers: %v", err))
	}
	if len(containers) == 0 {
		return cmd.Respond("📦 No containers with label `koncord.enable=true` found.")
	}

	var sb strings.Builder
	sb.WriteString("📦 **Managed Containers**\n")
	for _, c := range containers {
		sb.WriteString(fmt.Sprintf("• `%s` — %s (%s) — **%s**\n", c.Name, c.Image, c.ID, c.State))
	}
	return cmd.Respond(sb.String())
}

func (b *Bot) handleStatus(ctx context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleUser) {
		return cmd.Respond("🚫 You are not registered.")
	}

	name, err := b.resolveContainer(cmd)
	if err != nil {
		return cmd.Respond(err.Error())
	}

	ctr, err := b.runtime.GetContainer(ctx, name)
	if err != nil {
		return cmd.Respond(fmt.Sprintf("❌ %v", err))
	}

	msg := fmt.Sprintf(
		"📋 **%s**\n• Image: `%s`\n• ID: `%s`\n• Status: **%s**\n• State: %s",
		ctr.Name, ctr.Image, ctr.ID, ctr.Status, ctr.State,
	)
	return cmd.Respond(msg)
}

func (b *Bot) handleRestart(ctx context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleUser) {
		return cmd.Respond("🚫 You are not registered.")
	}

	name, err := b.resolveContainer(cmd)
	if err != nil {
		return cmd.Respond(err.Error())
	}

	if err := b.runtime.RestartContainer(ctx, name); err != nil {
		return cmd.Respond(fmt.Sprintf("❌ Failed to restart `%s`: %v", name, err))
	}
	return cmd.Respond(fmt.Sprintf("🔄 Container `%s` restarted successfully.", name))
}

func (b *Bot) handleLogs(ctx context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleUser) {
		return cmd.Respond("🚫 You are not registered.")
	}

	name, err := b.resolveContainer(cmd)
	if err != nil {
		return cmd.Respond(err.Error())
	}

	// Stop any existing log stream for this channel.
	b.stopLogForChannel(cmd.ChannelID)

	logCtx, cancel := context.WithCancel(context.Background())
	b.logMu.Lock()
	b.logCancel[cmd.ChannelID] = cancel
	b.logMu.Unlock()

	reader, err := b.runtime.StreamLogs(logCtx, name, time.Now())
	if err != nil {
		cancel()
		return cmd.Respond(fmt.Sprintf("❌ Failed to stream logs for `%s`: %v", name, err))
	}

	_ = cmd.Respond(fmt.Sprintf("📜 Now forwarding logs from `%s` to this channel. Use `/koncord logs` again to refresh.", name))

	// Stream in background.
	go func() {
		defer reader.Close()
		defer func() {
			b.logMu.Lock()
			delete(b.logCancel, cmd.ChannelID)
			b.logMu.Unlock()
		}()

		scanner := bufio.NewScanner(reader)
		var buf strings.Builder
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-logCtx.Done():
				return
			case <-ticker.C:
				if buf.Len() > 0 {
					msg := "```\n" + buf.String() + "```"
					if err := b.platform.SendMessage(logCtx, cmd.ChannelID, msg); err != nil {
						b.logger.Error("send log message", "err", err)
					}
					buf.Reset()
				}
			default:
				if scanner.Scan() {
					line := scanner.Text()
					// Docker log stream includes 8-byte header; strip it.
					if len(line) > 8 {
						line = line[8:]
					}
					buf.WriteString(line + "\n")
					// Flush if buffer is getting large.
					if buf.Len() > 1500 {
						msg := "```\n" + buf.String() + "```"
						if err := b.platform.SendMessage(logCtx, cmd.ChannelID, msg); err != nil {
							b.logger.Error("send log message", "err", err)
						}
						buf.Reset()
					}
				} else {
					// Scanner done (container stopped or error).
					if buf.Len() > 0 {
						msg := "```\n" + buf.String() + "```"
						_ = b.platform.SendMessage(logCtx, cmd.ChannelID, msg)
					}
					_ = b.platform.SendMessage(logCtx, cmd.ChannelID, "📜 Log stream ended.")
					return
				}
			}
		}
	}()

	return nil
}

func (b *Bot) handleAdminRegister(_ context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleAdmin) {
		return cmd.Respond("🚫 You need admin permissions for this command.")
	}

	targetID := cmd.Args["user"]
	if targetID == "" {
		return cmd.Respond("❌ Please mention a user.")
	}

	if err := b.auth.Register(targetID, domain.RoleAdmin); err != nil {
		return cmd.Respond(fmt.Sprintf("❌ Failed to register admin: %v", err))
	}
	return cmd.Respond(fmt.Sprintf("✅ <@%s> is now an **admin**.", targetID))
}

func (b *Bot) handleUserRegister(_ context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleAdmin) {
		return cmd.Respond("🚫 You need admin permissions for this command.")
	}

	targetID := cmd.Args["user"]
	if targetID == "" {
		return cmd.Respond("❌ Please mention a user.")
	}

	if err := b.auth.Register(targetID, domain.RoleUser); err != nil {
		return cmd.Respond(fmt.Sprintf("❌ Failed to register user: %v", err))
	}
	return cmd.Respond(fmt.Sprintf("✅ <@%s> is now a registered **user**.", targetID))
}

func (b *Bot) handleBind(_ context.Context, cmd chat.CommandContext) error {
	if !b.auth.Authorise(cmd.UserID, domain.RoleAdmin) {
		return cmd.Respond("🚫 You need admin permissions for this command.")
	}

	containerName := cmd.Args["container"]
	if containerName == "" {
		return cmd.Respond("❌ Please provide a container name.")
	}

	if err := b.store.Bind(cmd.ChannelID, containerName); err != nil {
		return cmd.Respond(fmt.Sprintf("❌ Failed to bind: %v", err))
	}
	return cmd.Respond(fmt.Sprintf("🔗 Container `%s` is now bound to this channel.", containerName))
}

// ---------- helpers ----------

// resolveContainer returns the container name from the command args, or falls
// back to the channel binding.
func (b *Bot) resolveContainer(cmd chat.CommandContext) (string, error) {
	if name, ok := cmd.Args["container"]; ok && name != "" {
		return name, nil
	}
	bound := b.store.GetBinding(cmd.ChannelID)
	if bound != "" {
		return bound, nil
	}
	return "", fmt.Errorf("❌ No container specified and no container is bound to this channel. Use `/koncord bind <container>` first.")
}

func (b *Bot) stopLogForChannel(channelID string) {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	if cancel, ok := b.logCancel[channelID]; ok {
		cancel()
		delete(b.logCancel, channelID)
	}
}
