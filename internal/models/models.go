package models

import "time"

// Message types
const (
	MsgTypeChat   = "chat"
	MsgTypeSystem = "system"
	MsgTypeJoin   = "join"
	MsgTypeLeave  = "leave"
	MsgTypeInfo   = "info"
)

// Message represents a chat message
type Message struct {
	Type     string    `json:"type"`
	Content  string    `json:"content"`
	Room     string    `json:"room"`
	Username string    `json:"username"`
	Time     time.Time `json:"time"`
}

// RoomInfo represents public info about a room.
// UserCount is not tracked in Compute@Edge and is omitted when zero.
type RoomInfo struct {
	ID        string `json:"id"`
	UserCount int    `json:"user_count,omitempty"`
	MsgCount  int    `json:"msg_count"`
}

// JoinRequest is the payload for joining a room
type JoinRequest struct {
	Room     string `json:"room"`
	Username string `json:"username"`
}

// ChatRequest is the payload for sending a message
type ChatRequest struct {
	Content string `json:"content"`
}

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
