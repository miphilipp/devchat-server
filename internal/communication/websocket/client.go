package websocket

import (
	//"fmt"
	"sync"
	"math/rand"
	"github.com/gorilla/websocket"
)

type client struct {
	id int
	conns []*websocket.Conn
	connsLock sync.Mutex
	Send chan messageFrame
	ReadClose chan struct{}
	Disconnect chan int
}
 
func (s *Server) DisconnectClient(clientID int)  {
	clients.RLock()
	client, ok := clients.m[clientID]
	clients.RUnlock()
	if ok {
		client.Disconnect <- websocket.CloseGoingAway
	}
}

func newClient(id int) *client {
	return &client{
		id: id,
		conns: make([]*websocket.Conn, 0, 2),
		Send: make(chan messageFrame),
		Disconnect: make(chan int),
		ReadClose: make(chan struct{}),
	}
}


// SendToClient sends a message over all the connections a client has established.
func (s *Server) SendToClient(clientID, source, id int, command RESTCommand, payload interface{})  {
	clients.RLock()
	client, ok := clients.m[clientID]
	clients.RUnlock()
	if ok {
		if id == 0 {
			id = rand.Int()
		}
		client.Send <- newFrame(source, id, command, payload)
	}
}

func (c *client) breakConnection(connection *websocket.Conn) bool {
	connectionIndex := -1

	c.connsLock.Lock()
	defer c.connsLock.Unlock()

	for i, conn := range c.conns {
		if conn == connection {
			connectionIndex = i
			break
		}
	}

	if connectionIndex == -1 {
		return len(c.conns) == 0
	}

	c.conns[connectionIndex] = c.conns[len(c.conns)-1]
	c.conns = c.conns[:len(c.conns)-1]
	return len(c.conns) == 0
}