package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DiscordToken  string
	GuildID       string // optional – restrict commands to a single guild
	StateFilePath string // path to the JSON state file
	LogLevel      slog.Level
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable is required")
	}

	stateFile := os.Getenv("KONCORD_STATE_FILE")
	if stateFile == "" {
		stateFile = "koncord_state.json"
	}

	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	return &Config{
		DiscordToken:  token,
		GuildID:       os.Getenv("DISCORD_GUILD_ID"),
		StateFilePath: stateFile,
		LogLevel:      level,
	}, nil
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
