package models

import (
	"encoding/json"
	"testing"
)

func TestMessageJSON(t *testing.T) {
	msg := Message{
		Type:     MsgTypeChat,
		Content:  "Hello world",
		Room:     "general",
		Username: "alice",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != MsgTypeChat {
		t.Errorf("expected type %s, got %s", MsgTypeChat, decoded.Type)
	}
	if decoded.Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got '%s'", decoded.Content)
	}
	if decoded.Room != "general" {
		t.Errorf("expected room 'general', got '%s'", decoded.Room)
	}
	if decoded.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", decoded.Username)
	}
}

func TestMessageTypeConstants(t *testing.T) {
	if MsgTypeChat != "chat" {
		t.Errorf("expected MsgTypeChat='chat', got '%s'", MsgTypeChat)
	}
	if MsgTypeSystem != "system" {
		t.Errorf("expected MsgTypeSystem='system', got '%s'", MsgTypeSystem)
	}
	if MsgTypeJoin != "join" {
		t.Errorf("expected MsgTypeJoin='join', got '%s'", MsgTypeJoin)
	}
	if MsgTypeLeave != "leave" {
		t.Errorf("expected MsgTypeLeave='leave', got '%s'", MsgTypeLeave)
	}
	if MsgTypeInfo != "info" {
		t.Errorf("expected MsgTypeInfo='info', got '%s'", MsgTypeInfo)
	}
}

func TestRoomInfoJSON(t *testing.T) {
	info := RoomInfo{
		ID:        "general",
		UserCount: 5,
		MsgCount:  100,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RoomInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != "general" {
		t.Errorf("expected ID 'general', got '%s'", decoded.ID)
	}
	if decoded.UserCount != 5 {
		t.Errorf("expected UserCount 5, got %d", decoded.UserCount)
	}
	if decoded.MsgCount != 100 {
		t.Errorf("expected MsgCount 100, got %d", decoded.MsgCount)
	}
}

func TestAPIResponseJSON(t *testing.T) {
	resp := APIResponse{
		Success: true,
		Data:    map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded APIResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Success != true {
		t.Error("expected Success=true")
	}
}

func TestAPIResponseError(t *testing.T) {
	resp := APIResponse{
		Success: false,
		Error:   "something went wrong",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded APIResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got '%s'", decoded.Error)
	}
}

func TestChatRequestJSON(t *testing.T) {
	req := ChatRequest{Content: "Hello!"}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ChatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got '%s'", decoded.Content)
	}
}
