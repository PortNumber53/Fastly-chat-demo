package fanout

import (
	"testing"

	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

func TestNew(t *testing.T) {
	p := New(true, "http://api.fanout.io/realm/test?gs=token123", "key", "domain")
	if p == nil {
		t.Fatal("expected non-nil publisher")
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		gripURL string
		want    bool
	}{
		{"enabled with URL", true, "http://api.fanout.io/realm/test?gs=token", true},
		{"enabled without URL", true, "", false},
		{"disabled with URL", false, "http://api.fanout.io/realm/test?gs=token", false},
		{"disabled without URL", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.enabled, tt.gripURL, "key", "domain")
			if p.IsEnabled() != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", p.IsEnabled(), tt.want)
			}
		})
	}
}

func TestPublishDisabled(t *testing.T) {
	p := New(false, "", "", "")
	msg := models.Message{
		Type:    models.MsgTypeChat,
		Content: "hello",
		Room:    "general",
	}
	err := p.Publish("general", msg)
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
}

func TestPublishInvalidGRIPURL(t *testing.T) {
	p := New(true, "not-a-valid-url", "key", "domain")
	msg := models.Message{
		Type:    models.MsgTypeChat,
		Content: "hello",
		Room:    "general",
	}
	err := p.Publish("general", msg)
	if err == nil {
		t.Error("expected error for invalid GRIP URL")
	}
}

func TestParseGRIPURL(t *testing.T) {
	p := New(true, "http://api.fanout.io/realm/myrealm?gs=mytoken", "key", "domain")

	publishURL, realm, token := p.parseGRIPURL()
	if publishURL != "http://api.fanout.io/realm" {
		t.Errorf("expected 'http://api.fanout.io/realm', got '%s'", publishURL)
	}
	if realm != "myrealm" {
		t.Errorf("expected 'myrealm', got '%s'", realm)
	}
	if token != "mytoken" {
		t.Errorf("expected 'mytoken', got '%s'", token)
	}
}

func TestParseGRIPURLInvalid(t *testing.T) {
	p := New(true, "", "", "")
	publishURL, realm, token := p.parseGRIPURL()
	// Empty URL returns empty values per parseGRIPURL's error handling
	if realm != "" || token != "" {
		t.Errorf("expected empty realm/token for empty URL, got realm='%s', token='%s'", realm, token)
	}
	_ = publishURL // publishURL may be non-empty for empty string parse
}

func TestPublishViaLogStream(t *testing.T) {
	p := New(true, "http://api.fanout.io/realm/test?gs=token", "key", "domain")
	msg := models.Message{
		Type:    models.MsgTypeChat,
		Content: "hello",
		Room:    "general",
		Username: "alice",
	}

	result, err := p.PublishViaLogStream("general", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// Should contain channel name
	if !contains(result, "chat-general") {
		t.Errorf("expected result to contain 'chat-general', got '%s'", result)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
				return true
			}
	}
	return false
}
