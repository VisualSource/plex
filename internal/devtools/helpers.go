package devtools

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

func readPacket(reader *bufio.Reader) (map[string]any, error) {
	lenBytes, err := reader.ReadSlice(':')
	if err != nil {
		return nil, err
	}
	len, err := strconv.Atoi(string(lenBytes[:len(lenBytes)-1])) // strip : from request
	if err != nil {
		return nil, err
	}
	if len < 0 || len > 16000000 /*16 MiB*/ {
		return nil, errors.New("packet size out of range")
	}

	payload := make([]byte, len)

	_, err = io.ReadFull(reader, payload)
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

	return writer.Flush()
}
