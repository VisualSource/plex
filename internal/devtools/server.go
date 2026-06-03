package devtools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
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

func readPacket(reader *bufio.Reader) (map[string]any, error) {
	lenBytes, err := reader.ReadSlice(':')
	if err != nil {
		return nil, err
	}
	len, err := strconv.Atoi(string(lenBytes))
	if err != nil {
		return nil, err
	}
	if len < 0 || len > 16000000 /*16 MiB*/ {
		return nil, errors.New("packet size out of range")
	}

	payload := make([]byte, len)
	_, err = reader.Read(payload)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {

		return nil, err
	}
	return data, nil
}

func writePacket(writer *bufio.Writer, data map[string]any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	size := len(bytes)

	packet := make([]byte, 0)
	packet = strconv.AppendInt(packet, int64(size), 10)
	packet = append(packet, ':')
	packet = append(packet, bytes...)

	_, err = writer.Write(packet)
	if err != nil {
		return err
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
