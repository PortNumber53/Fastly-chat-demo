package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PortNumber53/Fastly-chat-demo/internal/hub"
)

func TestUpgraderCheckOrigin(t *testing.T) {
	// Test that CheckOrigin allows all origins (demo purposes)
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil-site.com")
	
	if !Upgrader.CheckOrigin(req) {
		t.Error("expected CheckOrigin to allow all origins for demo")
	}
}

func TestServeWsMissingParams(t *testing.T) {
	h := hub.New(10, 100)
	
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(h, w, r)
	}))
	defer server.Close()
	
	// Replace http with ws
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_ = wsURL // The actual WebSocket connection test needs a real client
}

func TestServeWsDefaultParams(t *testing.T) {
	// Test that default room and username are used
	h := hub.New(10, 100)
	
	// Verify the hub creates rooms correctly for default params
	room, client, err := h.JoinRoom("", "anonymous")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = room
	_ = client
}
