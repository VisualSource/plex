package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/VisualSource/plex/internal/core"
	"github.com/VisualSource/plex/internal/log"
)

func main() {
	// Logger configuration
	logger := log.New(
		log.WithLevel(os.Getenv("LOG_LEVEL")),
		log.WithSource(),
	)

	if err := core.StartPlex(logger); err != nil {
		logger.ErrorContext(context.Background(), "an error occurred", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
