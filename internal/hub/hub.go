package hub

import (
	"sync"
	"time"

	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

// Hub maintains the set of active chat rooms and clients
type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]*Room
	maxRooms    int
	maxMessages int

	// Callback for Fastly Fanout integration
	OnBroadcast func(roomID string, msg models.Message)
}

// Room represents a single chat room
type Room struct {
	mu      sync.RWMutex
	id      string
	clients map[*Client]bool
	history []models.Message
}

// Client represents a connected WebSocket client
type Client struct {
	Hub      *Hub
	Room     *Room
	Send     chan models.Message
	Username string
}

// New creates a new Hub instance
func New(maxRooms, maxMessages int) *Hub {
	return &Hub{
		rooms:       make(map[string]*Room),
		maxRooms:    maxRooms,
		maxMessages: maxMessages,
	}
}

// ID returns the room's identifier
func (r *Room) ID() string {
	return r.id
}

// JoinRoom adds a client to a room (creates the room if needed)
func (h *Hub) JoinRoom(roomID, username string) (*Room, *Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[roomID]
	if !exists {
		if len(h.rooms) >= h.maxRooms {
			return nil, nil, ErrMaxRooms
		}
		room = &Room{
			id:      roomID,
			clients: make(map[*Client]bool),
			history: make([]models.Message, 0),
		}
		h.rooms[roomID] = room
	}

	client := &Client{
		Hub:      h,
		Room:     room,
		Send:     make(chan models.Message, 256),
		Username: username,
	}

	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	return room, client, nil
}

// LeaveRoom removes a client from their room
func (h *Hub) LeaveRoom(client *Client) {
	if client.Room == nil {
		return
	}

	room := client.Room
	room.mu.Lock()
	delete(room.clients, client)
	close(client.Send)
	room.mu.Unlock()

	// Broadcast leave message
	leaveMsg := models.Message{
		Type:     models.MsgTypeSystem,
		Content:  client.Username + " left the chat",
		Room:     room.id,
		Username: "system",
		Time:     time.Now(),
	}

	h.Broadcast(room.id, leaveMsg)

	// Clean up empty rooms
	h.mu.Lock()
	room.mu.RLock()
	empty := len(room.clients) == 0
	room.mu.RUnlock()
	if empty {
		delete(h.rooms, room.id)
	}
	h.mu.Unlock()
}

// Broadcast sends a message to all clients in a room
func (h *Hub) Broadcast(roomID string, msg models.Message) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Store in history
	room.mu.Lock()
	room.history = append(room.history, msg)
	if len(room.history) > h.maxMessages {
		room.history = room.history[len(room.history)-h.maxMessages:]
	}
	room.mu.Unlock()

	// Send to local clients
	room.mu.RLock()
	for client := range room.clients {
		select {
		case client.Send <- msg:
		default:
			// Client buffer full, skip
		}
	}
	room.mu.RUnlock()

	// Callback for Fastly Fanout
	if h.OnBroadcast != nil {
		h.OnBroadcast(roomID, msg)
	}
}

// GetHistory returns recent messages for a room
func (h *Hub) GetHistory(roomID string, limit int) []models.Message {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return []models.Message{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if limit <= 0 || limit > len(room.history) {
		limit = len(room.history)
	}

	history := make([]models.Message, limit)
	copy(history, room.history[len(room.history)-limit:])
	return history
}

// GetRooms returns a list of all active rooms
func (h *Hub) GetRooms() []models.RoomInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rooms := make([]models.RoomInfo, 0, len(h.rooms))
	for id, room := range h.rooms {
		room.mu.RLock()
		rooms = append(rooms, models.RoomInfo{
			ID:        id,
			UserCount: len(room.clients),
			MsgCount:  len(room.history),
		})
		room.mu.RUnlock()
	}
	return rooms
}

// GetRoomCount returns the number of connected clients for a room
func (h *Hub) GetRoomCount(roomID string) int {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	return len(room.clients)
}
