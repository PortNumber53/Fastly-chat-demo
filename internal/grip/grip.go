package grip

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

// FormatLogEntry formats a message for Fanout real-time log streaming.
func FormatLogEntry(roomID string, msg models.Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	channel := fmt.Sprintf("chat-%s", roomID)

	entry := map[string]interface{}{
		"channel": channel,
		"formats": map[string]interface{}{
			"ws": map[string]interface{}{
				"action": map[string]interface{}{
					"type": "send",
					"body": string(data),
				},
			},
			"http-stream": map[string]interface{}{
				"action": map[string]interface{}{
					"type": "send",
					"body": fmt.Sprintf("data: %s\n\n", string(data)),
				},
			},
		},
	}

	result, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal grip entry: %w", err)
	}

	return string(result), nil
}

// PublishToFanout writes a GRIP-formatted entry to the log endpoint.
func PublishToFanout(w io.Writer, roomID string, msg models.Message) error {
	entry, err := FormatLogEntry(roomID, msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, entry)
	return err
}
