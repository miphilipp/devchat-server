package websocket

import (
	"context"
	"math/rand"
	"sync"

	"github.com/go-kit/kit/log/level"
)

type room struct {
	Clients    []*client
	ClientLock sync.Mutex
	ID         int
}

func newRoom(id int) *room {
	return &room{
		Clients: make([]*client, 0, 2),
		ID:      id,
	}
}

func (r *room) broadcast(message messageFrame) {
	r.ClientLock.Lock()
	for _, client := range r.Clients {
		client.Send <- message
	}
	r.ClientLock.Unlock()
}

// RemoveClientFromRoom removes the client with the spcified id from the room.
// If the client or the room doesn't exist this function does nothing.
func (s *Server) RemoveClientFromRoom(roomNumber int, userID int) {
	s.rooms.RLock()
	room, ok := s.rooms.m[roomNumber]
	s.rooms.RUnlock()
	if !ok {
		return
	}

	room.ClientLock.Lock()
	defer room.ClientLock.Unlock()

	clientIndex := -1
	for i, c := range room.Clients {
		if c.id == userID {
			clientIndex = i
			break
		}
	}

	if clientIndex == -1 {
		return
	}

	room.Clients[clientIndex] = room.Clients[len(room.Clients)-1]
	room.Clients = room.Clients[:len(room.Clients)-1]
}

// AddRoom adds a new room to the room map.
func (s *Server) AddRoom(roomNumber int, initialClientID int) {
	s.rooms.RLock()
	_, ok := s.rooms.m[roomNumber]
	s.rooms.RUnlock()
	if ok {
		return
	}

	room := newRoom(roomNumber)
	s.rooms.Lock()
	s.rooms.m[roomNumber] = room
	s.rooms.Unlock()

	clients.RLock()
	client, ok := clients.m[initialClientID]
	clients.RUnlock()
	if ok {
		room.ClientLock.Lock()
		room.Clients = append(room.Clients, client)
		room.ClientLock.Unlock()
	}
}

func (s *Server) RemoveRoom(roomNumber int) {
	s.rooms.Lock()
	delete(s.rooms.m, roomNumber)
	s.rooms.Unlock()
}

// JoinRoom adds a client to a room
// If either the room or the client does not exist, nothing is done.
func (s *Server) JoinRoom(roomNumber int, userID int) {
	s.rooms.RLock()
	room, ok := s.rooms.m[roomNumber]
	s.rooms.RUnlock()
	if !ok {
		return
	}

	clients.RLock()
	client, ok := clients.m[userID]
	clients.RUnlock()
	if !ok {
		return
	}

	room.ClientLock.Lock()
	room.Clients = append(room.Clients, client)
	room.ClientLock.Unlock()
}

// BroadcastToRoom sends a RESTCommand with payload to every member of the
// specified room.
func (s *Server) BroadcastToRoom(roomNumber int, payload interface{}, ctx context.Context) {
	command := ctx.Value("command").(RESTCommand)
	id := ctx.Value("id").(int)
	if id == -1 {
		id = rand.Int()
	}

	s.rooms.RLock()
	room, ok := s.rooms.m[roomNumber]
	s.rooms.RUnlock()
	if ok {
		room.broadcast(newFrame(roomNumber, id, command, payload))
	} else {
		level.Warn(s.logger).Log(
			"Function", "BroadcastToRoom",
			"roomNumber", roomNumber,
			"err", "No such room")
	}
}
