package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := slog.New(h)
	slog.SetDefault(l)
	return l
}
