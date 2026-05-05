package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/rtlog"

	"github.com/PortNumber53/Fastly-chat-demo/internal/grip"
	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
	"github.com/PortNumber53/Fastly-chat-demo/internal/state"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(fsthttp.StatusNoContent)
			return
		}

		// Initialize state (KV store)
		st, err := state.New("chat_state")
		if err != nil {
			log.Printf("[State] KV store error: %v", err)
			fsthttp.Error(w, "service unavailable", fsthttp.StatusInternalServerError)
			return
		}

		// Log endpoint for Fanout publishing
		var fanoutLog *rtlog.Endpoint
		if ep := rtlog.Open("fanout"); ep != nil {
			fanoutLog = ep
		}

		path := r.URL.Path

		switch {
		case path == "/api/health":
			handleHealth(w, r, st)

		case path == "/api/rooms":
			handleRooms(w, r, st)

		case strings.HasPrefix(path, "/api/rooms/"):
			handleRoomDetail(w, r, st, fanoutLog)

		case path == "/ws":
			handleWebSocket(w, r, st, fanoutLog)

		default:
			serveStatic(w, r, path)
		}
	})
}

func handleHealth(w fsthttp.ResponseWriter, r *fsthttp.Request, st *state.State) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(fsthttp.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","version":"1.0.0","mode":"fastly-compute"}`)
}

func handleRooms(w fsthttp.ResponseWriter, r *fsthttp.Request, st *state.State) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(fsthttp.StatusOK)

	rooms, _ := st.GetRooms()
	resp, _ := json.Marshal(models.APIResponse{Success: true, Data: rooms})
	w.Write(resp)
}

func handleRoomDetail(w fsthttp.ResponseWriter, r *fsthttp.Request, st *state.State, fanoutLog *rtlog.Endpoint) {
	w.Header().Set("Content-Type", "application/json")

	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	if roomID == "" {
		w.WriteHeader(fsthttp.StatusBadRequest)
		fmt.Fprintf(w, `{"success":false,"error":"room id required"}`)
		return
	}

	switch r.Method {
	case "GET":
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}

		history, _ := st.GetRoomHistory(roomID, limit)
		resp, _ := json.Marshal(models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"room":       roomID,
				"user_count": 0,
				"messages":   history,
			},
		})
		w.WriteHeader(fsthttp.StatusOK)
		w.Write(resp)

	case "POST":
		username := r.URL.Query().Get("username")
		if username == "" {
			username = "api-user"
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(fsthttp.StatusBadRequest)
			fmt.Fprintf(w, `{"success":false,"error":"invalid body"}`)
			return
		}

		var req models.ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			req.Content = string(body)
		}

		if req.Content == "" {
			w.WriteHeader(fsthttp.StatusBadRequest)
			fmt.Fprintf(w, `{"success":false,"error":"empty message"}`)
			return
		}

		if len(req.Content) > 2000 {
			req.Content = req.Content[:2000]
		}

		msg := models.Message{
			Type:     models.MsgTypeChat,
			Content:  req.Content,
			Room:     roomID,
			Username: username,
			Time:     time.Now(),
		}

		// Ensure room is tracked
		if err := st.AddRoom(roomID); err != nil {
			log.Printf("[State] add room error: %v", err)
		}

		// Store in KV
		if err := st.AppendMessage(roomID, msg); err != nil {
			log.Printf("[State] append error: %v", err)
			w.WriteHeader(fsthttp.StatusInternalServerError)
			fmt.Fprintf(w, `{"success":false,"error":"failed to persist message"}`)
			return
		}

		// Publish to Fanout
		if fanoutLog != nil {
			if err := grip.PublishToFanout(fanoutLog, roomID, msg); err != nil {
				log.Printf("[Fanout] publish error: %v", err)
			}
		}

		w.WriteHeader(fsthttp.StatusOK)
		resp, _ := json.Marshal(models.APIResponse{Success: true, Data: msg})
		w.Write(resp)

	default:
		w.WriteHeader(fsthttp.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"success":false,"error":"method not allowed"}`)
	}
}

func handleWebSocket(w fsthttp.ResponseWriter, r *fsthttp.Request, st *state.State, fanoutLog *rtlog.Endpoint) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		roomID = "general"
	}
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "anonymous"
	}

	// Ensure room is tracked
	st.AddRoom(roomID)

	// Return GRIP headers so Fanout holds this connection
	channel := fmt.Sprintf("chat-%s", roomID)
	w.Header().Set("Grip-Hold", "stream")
	w.Header().Set("Grip-Channel", channel)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(fsthttp.StatusOK)

	// Publish join message
	if fanoutLog != nil {
		joinMsg := models.Message{
			Type:     models.MsgTypeJoin,
			Content:  username + " joined the chat",
			Room:     roomID,
			Username: username,
			Time:     time.Now(),
		}
		if err := grip.PublishToFanout(fanoutLog, roomID, joinMsg); err != nil {
			log.Printf("[Fanout] join publish error: %v", err)
		}
	}
}

func serveStatic(w fsthttp.ResponseWriter, r *fsthttp.Request, path string) {
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		fsthttp.Error(w, "static files error", fsthttp.StatusInternalServerError)
		return
	}

	filePath := strings.TrimPrefix(path, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	data, err := fs.ReadFile(staticSub, filePath)
	if err != nil {
		fsthttp.Error(w, "not found", fsthttp.StatusNotFound)
		return
	}

	contentType := "text/plain"
	switch {
	case strings.HasSuffix(filePath, ".html"):
		contentType = "text/html"
	case strings.HasSuffix(filePath, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(filePath, ".js"):
		contentType = "application/javascript"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(fsthttp.StatusOK)
	w.Write(data)
}
