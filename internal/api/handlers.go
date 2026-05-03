package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/PortNumber53/Fastly-chat-demo/internal/hub"
	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

// Handler holds dependencies for HTTP API handlers
type Handler struct {
	hub *hub.Hub
}

// New creates a new API handler
func New(h *hub.Hub) *Handler {
	return &Handler{hub: h}
}

// RegisterRoutes registers API routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/rooms", h.handleRooms)
	mux.HandleFunc("/api/rooms/", h.handleRoomDetail)
	mux.HandleFunc("/api/health", h.handleHealth)
}

// handleRooms lists all active rooms (GET)
func (h *Handler) handleRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case http.MethodGet:
		rooms := h.hub.GetRooms()
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data:    rooms,
		})
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "method not allowed",
		})
	}
}

// handleRoomDetail returns info/history for a specific room
func (h *Handler) handleRoomDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	if roomID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "room id required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		limitStr := r.URL.Query().Get("limit")
		limit := 50
		if limitStr != "" {
			if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
				limit = n
			}
		}

		history := h.hub.GetHistory(roomID, limit)
		userCount := h.hub.GetRoomCount(roomID)

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"room":       roomID,
				"user_count": userCount,
				"messages":   history,
			},
		})

	case http.MethodPost:
		// Send a message to a room via REST (for testing / simple integrations)
		var chatReq models.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "invalid request body",
			})
			return
		}

		username := r.URL.Query().Get("username")
		if username == "" {
			username = "api-user"
		}

		// Ensure the room exists (auto-create for REST API)
		if h.hub.GetRoomCount(roomID) == 0 {
			h.hub.JoinRoom(roomID, username)
		}

		msg := models.Message{
			Type:     models.MsgTypeChat,
			Content:  chatReq.Content,
			Room:     roomID,
			Username: username,
			Time:     parseTimeNow(),
		}

		h.hub.Broadcast(roomID, msg)
		log.Printf("[API] Message sent to room %s by %s", roomID, username)

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data:    msg,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "method not allowed",
		})
	}
}

// handleHealth returns service health information
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rooms := h.hub.GetRooms()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"rooms":   len(rooms),
	})
}
