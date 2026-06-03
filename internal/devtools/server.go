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

	if err := writePacket(writer, map[string]any{
		"from":            "root",
		"applicationType": "browser",
		"traits": map[string]bool{
			"networkMonitor": false,
		},
	}); err != nil {
		logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
		return
	}

	logger.DebugContext(ctx, "new connection", slog.String("remote", conn.RemoteAddr().String()))

	for {
		message, err := readPacket(reader)
		if err != nil {
			logger.ErrorContext(ctx, "packet read error", slog.String("error", err.Error()))
			continue
		}

		logger.DebugContext(ctx, "(server) packet", slog.Any("packet", message))

		switch message["to"] {
		case "root":

		default:
			if err := writePacket(writer, map[string]any{
				"from":    message["to"],
				"error":   "unrecognizedPacketType",
				"message": "unable to process request",
			}); err != nil {
				logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				return
			}

		}

	}
}
