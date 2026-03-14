package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ender-events/koncord/internal/auth"
	"github.com/Ender-events/koncord/internal/bot"
	"github.com/Ender-events/koncord/internal/chat"
	"github.com/Ender-events/koncord/internal/chat/discord"
	"github.com/Ender-events/koncord/internal/config"
	dockerrt "github.com/Ender-events/koncord/internal/runtime/docker"
	"github.com/Ender-events/koncord/internal/store"
)

func main() {
	// ── Config ──────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// ── Store ───────────────────────────────────────────────────────────
	st, err := store.New(cfg.StateFilePath)
	if err != nil {
		logger.Error("failed to open store", "err", err)
		os.Exit(1)
	}

	// ── Container runtime ───────────────────────────────────────────────
	rt, err := dockerrt.New(ctx)
	if err != nil {
		logger.Error("failed to create docker runtime", "err", err)
		os.Exit(1)
	}
	defer rt.Close()

	// ── Auth ────────────────────────────────────────────────────────────
	authMgr := auth.NewManager(st)

	// ── Bot (command router) — created first so we can pass its handler ─
	// We need the platform reference inside bot, but bot.HandleCommand is
	// needed to create the platform. We solve this with a two-phase init.
	var b *bot.Bot

	// ── Chat platform ───────────────────────────────────────────────────
	discordBot, err := discord.New(cfg.DiscordToken, cfg.GuildID, func(ctx context.Context, cmd chat.CommandContext) error {
		return b.HandleCommand(ctx, cmd)
	}, logger)
	if err != nil {
		logger.Error("failed to create discord bot", "err", err)
		os.Exit(1)
	}

	b = bot.New(authMgr, rt, discordBot, st, logger)

	// ── Start ───────────────────────────────────────────────────────────
	if err := discordBot.Start(ctx); err != nil {
		logger.Error("failed to start discord bot", "err", err)
		os.Exit(1)
	}

	logger.Info("koncord is running — press Ctrl+C to stop")

	// ── Wait for shutdown signal ────────────────────────────────────────
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("shutting down…")
	b.StopAllLogs()

	if err := discordBot.Stop(); err != nil {
		logger.Error("discord stop error", "err", err)
	}

	logger.Info("goodbye")
}
