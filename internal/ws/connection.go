package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PortNumber53/Fastly-chat-demo/internal/hub"
	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second
	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer
	maxMessageSize = 2048
)

// Upgrader upgrades HTTP connections to WebSocket
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for demo purposes
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWs handles WebSocket requests from peers
func ServeWs(h *hub.Hub, w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		roomID = "general"
	}
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "anonymous"
	}

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}

	_, client, err := h.JoinRoom(roomID, username)
	if err != nil {
		log.Printf("[WS] Join room error: %v", err)
		conn.Close()
		return
	}

	// Send room history FIRST (before join broadcast to avoid duplicate)
	history := h.GetHistory(roomID, 50)
	for _, msg := range history {
		select {
		case client.Send <- msg:
		default:
		}
	}

	// Then broadcast the join message
	joinMsg := models.Message{
		Type:     models.MsgTypeJoin,
		Content:  username + " joined the chat",
		Room:     roomID,
		Username: username,
		Time:     time.Now(),
	}
	h.Broadcast(roomID, joinMsg)

	go writePump(client, conn)
	go readPump(client, conn, h)
}

// readPump pumps messages from the WebSocket connection to the hub
func readPump(client *hub.Client, conn *websocket.Conn, h *hub.Hub) {
	defer func() {
		h.LeaveRoom(client)
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}

		var chatReq models.ChatRequest
		if err := json.Unmarshal(message, &chatReq); err != nil {
			// Try plain text message
			chatReq.Content = string(message)
		}

		if chatReq.Content == "" {
			continue
		}

		// Truncate overly long messages
		if len(chatReq.Content) > 2000 {
			chatReq.Content = chatReq.Content[:2000]
		}

		msg := models.Message{
			Type:     models.MsgTypeChat,
			Content:  chatReq.Content,
			Room:     client.Room.ID(),
			Username: client.Username,
			Time:     time.Now(),
		}

		h.Broadcast(client.Room.ID(), msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func writePump(client *hub.Client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("[WS] JSON marshal error: %v", err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
