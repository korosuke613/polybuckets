package internal

import (
	"log/slog"
	"os"
)

// NewJsonLogger creates a new JSON logger.
func NewJsonLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
}
