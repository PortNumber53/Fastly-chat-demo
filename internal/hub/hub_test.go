package hub

import (
	"testing"

	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

func TestNew(t *testing.T) {
	h := New(10, 100)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.maxRooms != 10 {
		t.Errorf("expected maxRooms=10, got %d", h.maxRooms)
	}
	if h.maxMessages != 100 {
		t.Errorf("expected maxMessages=100, got %d", h.maxMessages)
	}
}

func TestJoinRoom(t *testing.T) {
	h := New(10, 100)
	room, client, err := h.JoinRoom("general", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room == nil || client == nil {
		t.Fatal("expected non-nil room and client")
	}
	if room.ID() != "general" {
		t.Errorf("expected room ID 'general', got '%s'", room.ID())
	}
	if client.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", client.Username)
	}
}

func TestJoinRoomCreatesNew(t *testing.T) {
	h := New(10, 100)
	room1, _, _ := h.JoinRoom("room1", "alice")
	room2, _, _ := h.JoinRoom("room2", "bob")
	if room1.ID() == room2.ID() {
		t.Error("expected different room IDs")
	}
}

func TestJoinSameRoom(t *testing.T) {
	h := New(10, 100)
	room1, c1, _ := h.JoinRoom("general", "alice")
	room2, c2, _ := h.JoinRoom("general", "bob")
	if room1.ID() != room2.ID() {
		t.Error("expected same room for same room ID")
	}
	_ = c1
	_ = c2
}

func TestMaxRooms(t *testing.T) {
	h := New(2, 100)
	_, _, err1 := h.JoinRoom("room1", "a")
	_, _, err2 := h.JoinRoom("room2", "b")
	_, _, err3 := h.JoinRoom("room3", "c")
	if err1 != nil {
		t.Fatalf("unexpected error for room1: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("unexpected error for room2: %v", err2)
	}
	if err3 != ErrMaxRooms {
		t.Errorf("expected ErrMaxRooms, got %v", err3)
	}
}

func TestLeaveRoom(t *testing.T) {
	h := New(10, 100)
	var broadcastCalled bool
	h.OnBroadcast = func(roomID string, msg models.Message) {
		broadcastCalled = true
	}

	_, client, _ := h.JoinRoom("general", "alice")
	h.LeaveRoom(client)

	if !broadcastCalled {
		t.Error("expected broadcast to be called on leave")
	}

	// Room should be cleaned up since empty
	h.mu.RLock()
	_, exists := h.rooms["general"]
	h.mu.RUnlock()
	if exists {
		t.Error("expected room to be cleaned up after all clients leave")
	}
}

func TestLeaveRoomNil(t *testing.T) {
	h := New(10, 100)
	client := &Client{Hub: h, Room: nil, Username: "ghost"}
	// Should not panic
	h.LeaveRoom(client)
}

func TestBroadcast(t *testing.T) {
	h := New(10, 100)

	_, c1, _ := h.JoinRoom("general", "alice")
	_, c2, _ := h.JoinRoom("general", "bob")

	msg := models.Message{
		Type:    models.MsgTypeChat,
		Content: "hello",
		Room:    "general",
		Username: "alice",
	}

	h.Broadcast("general", msg)

	// Both clients should receive the message
	select {
	case m := <-c1.Send:
		if m.Content != "hello" {
			t.Errorf("expected 'hello', got '%s'", m.Content)
		}
	default:
		t.Error("client 1 should have received message")
	}

	select {
	case m := <-c2.Send:
		if m.Content != "hello" {
			t.Errorf("expected 'hello', got '%s'", m.Content)
		}
	default:
		t.Error("client 2 should have received message")
	}
}

func TestBroadcastCallback(t *testing.T) {
	h := New(10, 100)

	var callbackRoom string
	var callbackMsg models.Message
	h.OnBroadcast = func(roomID string, msg models.Message) {
		callbackRoom = roomID
		callbackMsg = msg
	}

	_, _, _ = h.JoinRoom("test", "user")
	msg := models.Message{
		Type:    models.MsgTypeChat,
		Content: "test msg",
		Room:    "test",
		Username: "user",
	}
	h.Broadcast("test", msg)

	if callbackRoom != "test" {
		t.Errorf("expected callback room 'test', got '%s'", callbackRoom)
	}
	if callbackMsg.Content != "test msg" {
		t.Errorf("expected callback content 'test msg', got '%s'", callbackMsg.Content)
	}
}

func TestBroadcastNonExistentRoom(t *testing.T) {
	h := New(10, 100)
	// Should not panic
	msg := models.Message{Type: models.MsgTypeChat, Content: "hello", Room: "no-room"}
	h.Broadcast("no-room", msg)
}

func TestGetHistory(t *testing.T) {
	h := New(10, 5) // max 5 messages

	_, client, _ := h.JoinRoom("general", "alice")

	// Broadcast 5 messages
	for i := 0; i < 5; i++ {
		h.Broadcast("general", models.Message{
			Type:    models.MsgTypeChat,
			Content: "msg",
			Room:    "general",
		})
	}

	// Drain client channel
	for len(client.Send) > 0 {
		<-client.Send
	}

	history := h.GetHistory("general", 0)
	if len(history) != 5 {
		t.Errorf("expected 5 messages, got %d", len(history))
	}
}

func TestGetHistoryTruncation(t *testing.T) {
	h := New(10, 3) // max 3 messages

	_, client, _ := h.JoinRoom("general", "alice")

	for i := 0; i < 5; i++ {
		h.Broadcast("general", models.Message{
			Type:    models.MsgTypeChat,
			Content: "msg",
			Room:    "general",
		})
	}

	for len(client.Send) > 0 {
		<-client.Send
	}

	history := h.GetHistory("general", 0)
	if len(history) != 3 {
		t.Errorf("expected 3 messages (truncated), got %d", len(history))
	}
}

func TestGetHistoryNonExistentRoom(t *testing.T) {
	h := New(10, 100)
	history := h.GetHistory("nosuchroom", 10)
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d messages", len(history))
	}
}

func TestGetHistoryLimit(t *testing.T) {
	h := New(10, 100)

	_, client, _ := h.JoinRoom("general", "alice")

	for i := 0; i < 10; i++ {
		h.Broadcast("general", models.Message{
			Type:    models.MsgTypeChat,
			Content: "msg",
			Room:    "general",
		})
	}

	for len(client.Send) > 0 {
		<-client.Send
	}

	history := h.GetHistory("general", 5)
	if len(history) != 5 {
		t.Errorf("expected 5 messages with limit, got %d", len(history))
	}
}

func TestGetRooms(t *testing.T) {
	h := New(10, 100)
	h.JoinRoom("room1", "a")
	h.JoinRoom("room2", "b")

	rooms := h.GetRooms()
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}
}

func TestGetRoomCount(t *testing.T) {
	h := New(10, 100)
	h.JoinRoom("general", "alice")
	h.JoinRoom("general", "bob")

	count := h.GetRoomCount("general")
	if count != 2 {
		t.Errorf("expected 2 users, got %d", count)
	}
}

func TestGetRoomCountNonExistent(t *testing.T) {
	h := New(10, 100)
	count := h.GetRoomCount("nosuchroom")
	if count != 0 {
		t.Errorf("expected 0 for non-existent room, got %d", count)
	}
}

func TestRoomID(t *testing.T) {
	h := New(10, 100)
	room, _, _ := h.JoinRoom("test-room", "user")
	if room.ID() != "test-room" {
		t.Errorf("expected 'test-room', got '%s'", room.ID())
	}
}
