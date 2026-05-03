package fanout

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

// Publisher handles publishing messages via Fastly Fanout / GRIP
// It can work in two modes:
// 1. Direct Fanout Cloud API (for self-hosted / non-Compute@Edge deployments)
// 2. Realtime Log Streaming (for Compute@Edge deployments)
type Publisher struct {
	enabled    bool
	gripURL    string
	gripKey    string
	domain     string
	httpClient *http.Client
}

// New creates a new Fanout publisher
func New(enabled bool, gripURL, gripKey, domain string) *Publisher {
	return &Publisher{
		enabled: enabled,
		gripURL: gripURL,
		gripKey: gripKey,
		domain:  domain,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsEnabled returns whether Fanout publishing is active
func (p *Publisher) IsEnabled() bool {
	return p.enabled && p.gripURL != ""
}

// Publish sends a message to all subscribers of a room via Fanout
// Uses the GRIP (Generic Realtime Intermediary Protocol) format
func (p *Publisher) Publish(roomID string, msg models.Message) error {
	if !p.IsEnabled() {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Build the GRIP publish body
	channel := fmt.Sprintf("chat-%s", roomID)

	// Parse the GRIP URL to get the publish endpoint
	publishURL, realm, gsToken := p.parseGRIPURL()
	if publishURL == "" {
		return fmt.Errorf("invalid GRIP URL")
	}

	// Build GRIP publish request
	body := map[string]interface{}{
		"items": []map[string]interface{}{{
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
						"type":    "send",
						"body":    fmt.Sprintf("data: %s\n\n", string(data)),
					},
				},
			},
		}},
	}

	bodyData, _ := json.Marshal(body)

	publishEndpoint := fmt.Sprintf("%s/%s", publishURL, realm)
	req, err := http.NewRequest("POST", publishEndpoint, strings.NewReader(string(bodyData)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gsToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("publish request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed with status %d", resp.StatusCode)
	}

	log.Printf("[Fanout] Published to channel %s", channel)
	return nil
}

// parseGRIPURL extracts the publish URL, realm, and token from the GRIP URL
func (p *Publisher) parseGRIPURL() (publishURL, realm, token string) {
	// Expected format: http://api.fanout.io/realm/REALM?gs=TOKEN
	u, err := url.Parse(p.gripURL)
	if err != nil {
		return "", "", ""
	}

	realm = strings.TrimPrefix(u.Path, "/realm/")
	token = u.Query().Get("gs")
	publishURL = fmt.Sprintf("%s://%s/realm", u.Scheme, u.Host)

	return publishURL, realm, token
}

// PublishViaLogStream formats a message for Fastly Realtime Log Streaming
// This is used when running on Compute@Edge where messages are published
// by writing to a log endpoint that Fanout subscribes to
func (p *Publisher) PublishViaLogStream(roomID string, msg models.Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	channel := fmt.Sprintf("chat-%s", roomID)

	// GRIP format for log streaming
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
					"body":    fmt.Sprintf("data: %s\n\n", string(data)),
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
