package state

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fastly/compute-sdk-go/kvstore"
	"github.com/PortNumber53/Fastly-chat-demo/internal/models"
)

const maxMessagesPerRoom = 100

// State manages chat room data backed by a Fastly KV store.
type State struct {
	store *kvstore.Store
}

// New opens the named KV store.
func New(name string) (*State, error) {
	s, err := kvstore.Open(name)
	if err != nil {
		return nil, err
	}
	return &State{store: s}, nil
}

// GetRoomHistory returns recent messages for a room.
func (s *State) GetRoomHistory(roomID string, limit int) ([]models.Message, error) {
	key := fmt.Sprintf("history:%s", roomID)
	v, err := s.store.Lookup(key)
	if err != nil {
		if err == kvstore.ErrKeyNotFound {
			return []models.Message{}, nil
		}
		return nil, err
	}

	data, err := io.ReadAll(v)
	if err != nil {
		return nil, err
	}

	var history []models.Message
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	if limit > 0 && limit < len(history) {
		return history[len(history)-limit:], nil
	}
	return history, nil
}

// AppendMessage adds a message to a room's history.
func (s *State) AppendMessage(roomID string, msg models.Message) error {
	history, err := s.GetRoomHistory(roomID, 0)
	if err != nil {
		return err
	}

	history = append(history, msg)
	if len(history) > maxMessagesPerRoom {
		history = history[len(history)-maxMessagesPerRoom:]
	}

	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("history:%s", roomID)
	return s.store.Insert(key, strings.NewReader(string(data)))
}

// AddRoom registers a room ID in the global room list.
func (s *State) AddRoom(roomID string) error {
	rooms, err := s.getRoomList()
	if err != nil {
		return err
	}
	for _, r := range rooms {
		if r == roomID {
			return nil
		}
	}
	rooms = append(rooms, roomID)
	data, err := json.Marshal(rooms)
	if err != nil {
		return err
	}
	return s.store.Insert("rooms", strings.NewReader(string(data)))
}

// GetRooms returns a list of known rooms.
func (s *State) GetRooms() ([]models.RoomInfo, error) {
	roomIDs, err := s.getRoomList()
	if err != nil {
		return nil, err
	}
	result := make([]models.RoomInfo, 0, len(roomIDs))
	for _, id := range roomIDs {
		result = append(result, models.RoomInfo{
			ID:        id,
			UserCount: 0,
			MsgCount:  0,
		})
	}
	return result, nil
}

func (s *State) getRoomList() ([]string, error) {
	v, err := s.store.Lookup("rooms")
	if err != nil {
		if err == kvstore.ErrKeyNotFound {
			return []string{}, nil
		}
		return nil, err
	}
	data, err := io.ReadAll(v)
	if err != nil {
		return nil, err
	}
	var rooms []string
	if err := json.Unmarshal(data, &rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}
