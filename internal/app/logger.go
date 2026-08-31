package app

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(format, level string) *slog.Logger {
	var handler slog.Handler
	options := &slog.HandlerOptions{Level: parseLogLevel(level)}
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}
	return slog.New(handler)
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
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
