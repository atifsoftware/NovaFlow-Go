package core

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger initializes the global slog default logger based on APP_ENV.
// If env is "production" or "staging", it configures a JSON handler.
// Otherwise, it uses a user-friendly Text handler.
func InitLogger(env string) {
	var handler slog.Handler
	level := slog.LevelInfo

	if env == "" {
		env = "local"
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	envLower := strings.ToLower(env)
	if envLower == "production" || envLower == "staging" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
