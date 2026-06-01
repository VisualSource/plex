package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	"github.com/VisualSource/plex/internal/core"
	"github.com/VisualSource/plex/internal/log"
)

var sLogLevelFlag = flag.String("logLevel", "", "set logger log level")

func main() {
	flag.Parse()

	// Logger configuration
	logger := log.New(
		log.WithLevel(*sLogLevelFlag),
		log.WithSource(),
	)

	ctx := context.Background()

	logger.DebugContext(ctx, "starting plex!")

	go func() {
		window := new(app.Window)

		window.Option(app.Title("Plex"))
		window.Option(app.Size(unit.Dp(854), unit.Dp(480)))

		if err := core.StartPlex(ctx, logger, window); err != nil {
			logger.ErrorContext(ctx, "window error", slog.Any("error", err))
		}

		os.Exit(0)
	}()

	app.Main()
}
