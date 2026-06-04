package devtools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
)

func StartRemoteDebuggingServer(logger *slog.Logger, ctx context.Context, port int) error {
	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.DebugContext(ctx, "starting remote devtools", slog.Int("port", port))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			logger.ErrorContext(ctx, "failed to accept connection", slog.String("error", err.Error()))
			continue
		}

		go handleConnection(logger, ctx, conn)
	}

	return nil
}

func handleConnection(logger *slog.Logger, ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	err := writePacket(writer, map[string]any{
		"from":            "root",
		"applicationType": "browser",
		"traits": map[string]bool{
			"networkMonitor": false,
		},
	})
	if err != nil {
		logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
		return
	}

	logger.DebugContext(ctx, "new connection", slog.String("remote", conn.RemoteAddr().String()))

	for {
		message, err := readPacket(reader)
		if err != nil {
			logger.ErrorContext(ctx, "packet read error", slog.String("error", err.Error()))
			return
		}

		logger.DebugContext(ctx, "read packet", slog.Any("packet", message))

		switch message["to"].(string) {
		case "root":
			switch message["type"].(string) {
			case "connect":
				if err := writePacket(writer, map[string]any{
					"from": "root",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getRoot":
				if err := writePacket(writer, map[string]any{
					"from":            "root",
					"selected":        0,
					"preferenceActor": "plex.conn0.pref",
					"deviceActor":     "plex.conn0.device1",
					"tabs": []map[string]any{
						{
							"actor": "plex.conn0.tabDescriptor1",
							"title": "Plex",
							"url":   "about:blank",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listTabs":
				if err := writePacket(writer, map[string]any{
					"from":     "root",
					"selected": 0,
					"tabs": []map[string]any{
						{
							"actor": "plex.conn0.tabDescriptor1",
							"title": "Plex",
							"url":   "about:blank",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.device1":
			switch message["type"].(string) {
			case "getDescription":
				if err := writePacket(writer, map[string]any{
					"from": "plex.conn0.device1",
					"value": map[string]any{
						"apptype":   "browser",
						"name":      "Plex",
						"vender":    "Plex",
						"brandName": "Plex",
						"version":   "0.1.0",
						"channel":   "release",
						"os":        "Linux",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				}
			}
		default:
			err := writePacket(writer, map[string]any{
				"from":    message["to"],
				"error":   "unrecognizedPacketType",
				"message": "unable to process request",
			})
			if err != nil {
				logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				return
			}
			logger.DebugContext(ctx, "sent packet")
		}

	}
}
