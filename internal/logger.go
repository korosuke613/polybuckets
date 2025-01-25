package internal

import (
	"log/slog"
	"os"
)

func NewJsonLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
}
