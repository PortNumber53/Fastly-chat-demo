package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/PortNumber53/Fastly-chat-demo/internal/api"
	"github.com/PortNumber53/Fastly-chat-demo/internal/fanout"
	"github.com/PortNumber53/Fastly-chat-demo/internal/hub"
	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
	"github.com/PortNumber53/Fastly-chat-demo/internal/ws"
	"github.com/PortNumber53/Fastly-chat-demo/pkg/config"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	cfg := config.Load()

	// Allow overrides via flags
	port := flag.String("port", cfg.Port, "listen port")
	host := flag.String("host", cfg.Host, "listen host")
	flag.Parse()

	// Initialize the chat hub
	chatHub := hub.New(cfg.MaxRooms, cfg.MaxMessagesPerRoom)

	// Initialize the Fastly Fanout publisher
	publisher := fanout.New(
		cfg.FanoutEnabled,
		cfg.GRIPURL,
		cfg.GRIPKey,
		cfg.FanoutDomain,
	)

	// Wire up the broadcast callback for Fanout publishing
	if publisher.IsEnabled() {
		chatHub.OnBroadcast = func(roomID string, msg models.Message) {
			if err := publisher.Publish(roomID, msg); err != nil {
				log.Printf("[Fanout] Publish error: %v", err)
			}
		}
		log.Println("[Fanout] Fastly Fanout publishing ENABLED")
	} else {
		chatHub.OnBroadcast = func(roomID string, msg models.Message) {
			// No-op when Fanout is disabled
		}
		log.Println("[Fanout] Fastly Fanout publishing disabled (local WebSocket mode)")
	}

	// Set up HTTP mux
	mux := http.NewServeMux()

	// Register API handlers
	handler := api.New(chatHub)
	handler.RegisterRoutes(mux)

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(chatHub, w, r)
	})

	// Serve static frontend files
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to access embedded static files: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	// Apply middleware
	var handlerChain http.Handler = mux
	handlerChain = corsMiddleware(handlerChain)
	handlerChain = loggingMiddleware(handlerChain)

	// Start server
	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("🚀 Fastly Chat Demo starting on %s", addr)
	log.Printf("   WebSocket:   ws://%s/ws", addr)
	log.Printf("   API:         http://%s/api/health", addr)
	log.Printf("   Rooms API:   http://%s/api/rooms", addr)
	log.Printf("   Fanout:      %v", publisher.IsEnabled())

	server := &http.Server{
		Addr:              addr,
		Handler:           handlerChain,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware adds CORS headers for cross-origin API access
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each request
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[HTTP] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
