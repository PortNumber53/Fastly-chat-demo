package hub

import "errors"

var (
	ErrMaxRooms   = errors.New("maximum number of rooms reached")
	ErrRoomFull   = errors.New("room is full")
	ErrBadMessage = errors.New("invalid message")
)
