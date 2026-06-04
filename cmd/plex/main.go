package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	"github.com/VisualSource/plex/internal/core"
	"github.com/VisualSource/plex/internal/devtools"
	"github.com/VisualSource/plex/internal/log"
)

var sLogLevelFlag = flag.String("logLevel", "", "set logger log level")
var iRemoteDebuggingPortFlag = flag.Int("remoteDebuggingPort", 0, "set remote debugging port")

func main() {
	flag.Parse()

	// Logger configuration
	logger := log.New(
		log.WithLevel(*sLogLevelFlag),
		log.WithSource(),
	)

	ctx, cancel := context.WithCancel(context.Background())

	logger.DebugContext(ctx, "starting plex!")

	if *iRemoteDebuggingPortFlag != 0 {
		go func() {
			if err := devtools.StartRemoteDebuggingServer(logger, ctx, *iRemoteDebuggingPortFlag); err != nil {
				logger.ErrorContext(ctx, "devtools error", slog.String("error", err.Error()))
			}
		}()
	}

	go func() {
		window := new(app.Window)

		window.Option(app.Title("Plex"))
		window.Option(app.Size(unit.Dp(720), unit.Dp(480)))

		if err := core.StartPlex(ctx, logger, window, *iRemoteDebuggingPortFlag); err != nil {
			logger.ErrorContext(ctx, "window error", slog.Any("error", err))
		}
		cancel()
		os.Exit(0)
	}()

	app.Main()
}
