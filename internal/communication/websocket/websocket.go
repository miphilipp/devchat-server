package websocket

import (
	"context"
	"encoding/json"
	"net/http"

	//"fmt"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/throttled/throttled"

	"github.com/gorilla/websocket"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/conversations"
	"github.com/miphilipp/devchat-server/internal/messaging"
	"github.com/miphilipp/devchat-server/internal/user"
)

var (
	clients = struct {
		sync.RWMutex
		m map[int]*client
	}{m: make(map[int]*client)}

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type endpoint struct {
	command   RESTCommand
	isLimited bool
	handler   func(ctx context.Context, clientID int, frame messageFrame) error
}

type Server struct {
	Messaging     messaging.Service
	Conversations conversations.Service
	User          user.Service

	logger  log.Logger
	limiter *WebsocketRateLimiter

	endpoints []endpoint

	rooms struct {
		sync.RWMutex
		m map[int]*room
	}
}

func New(
	messagingService messaging.Service,
	conversationService conversations.Service,
	userService user.Service,
	limiterStore throttled.GCRAStore,
	logger log.Logger) *Server {

	vary := &WebsocketVaryBy{RemoteAddr: true, Method: false, Ressource: true}
	limiter, err := newWebsocketRateLimiter(limiterStore, vary, 20, 3)
	if err != nil {
		return nil
	}

	server := &Server{
		Messaging:     messagingService,
		Conversations: conversationService,
		User:          userService,
		logger:        logger,
		limiter:       limiter,
		endpoints:     make([]endpoint, 0, 10),
		rooms: struct {
			sync.RWMutex
			m map[int]*room
		}{m: make(map[int]*room)},
	}

	conversations, err := server.Conversations.ListConversations()
	if err != nil {
		level.Error(logger).Log("err", err)
		return nil
	}

	for _, c := range conversations {
		server.rooms.m[c.ID] = newRoom(c.ID)
	}

	registerEndpoints(server)

	return server
}

func (s *Server) removeClientFromRooms(client *client) {
	s.rooms.RLock()
	for _, room := range s.rooms.m {
		s.rooms.RUnlock()
		s.RemoveClientFromRoom(room.ID, client.id)
		s.rooms.RLock()
	}
	s.rooms.RUnlock()
}

// StartWebsocket upgrades the connection to a websocket connection.
func (s *Server) StartWebsocket(w http.ResponseWriter, r *http.Request, user int) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		level.Error(s.logger).Log("err", err)
		return err
	}

	clients.RLock()
	c, ok := clients.m[user]
	clients.RUnlock()
	if !ok {
		c = newClient(user)

		clients.Lock()
		clients.m[user] = c
		clients.Unlock()

		conversationWithUser, err := s.Conversations.ListConversationsForUser(user)
		if err != nil {
			level.Error(s.logger).Log("err", err)
			return err
		}

		for _, conversation := range conversationWithUser {
			s.JoinRoom(conversation.ID, c.id)
		}

		s.User.UpdateOnlineTimestamp(c.id)

		go s.sendLoop(c)
	}

	conn.SetCloseHandler(func(code int, text string) error {
		s.cleanupAfterClient(conn, c)
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		return nil
	})

	c.connsLock.Lock()
	c.conns = append(c.conns, conn)
	c.connsLock.Unlock()

	go s.receiveLoop(conn, c)
	return nil
}

func (s *Server) cleanupAfterClient(conn *websocket.Conn, client *client) {
	isCompletlyDisconnected := client.breakConnection(conn)
	if isCompletlyDisconnected {
		clients.Lock()
		delete(clients.m, client.id)
		clients.Unlock()
		s.removeClientFromRooms(client)
		client.Disconnect <- 1000 // Code is not relevant here
	}
}

func (s *Server) sendLoop(client *client) {
	for {
		select {
		case msg := <-client.Send:
			for _, conn := range client.conns {
				err := conn.WriteJSON(msg)
				if err != nil {
					level.Error(s.logger).Log("Function", "sendLoop", "client", client.id, "err", err)
				}
			}
		case code := <-client.Disconnect:
			for _, conn := range client.conns {
				err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, ""))
				if err != nil {
					level.Error(s.logger).Log("Function", "sendLoop", "client", client.id, "err", err)
				}
			}
			return
		}
	}
}

func (s *Server) receiveLoop(conn *websocket.Conn, c *client) {
	defer conn.Close()

	for {
		var msg json.RawMessage
		wrapper := messageFrame{
			Payload: &msg,
		}

		err := conn.ReadJSON(&wrapper)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				level.Debug(s.logger).Log("err", err)
				s.cleanupAfterClient(conn, c)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseAbnormalClosure) {
				level.Info(s.logger).Log("message", "Connection has been closed by peer")
			} else {
				level.Error(s.logger).Log("err", err)
				s.cleanupAfterClient(conn, c)
			}
			return
		}

		// Ignore heartbeat
		if wrapper.Command.Method == HeartbeatCommandMethod {
			continue
		}

		shouldLimit, err := s.limiter.RateLimit(
			wrapper.Command.Method,
			wrapper.Command.Ressource,
			conn.UnderlyingConn().RemoteAddr().String(),
		)

		isFound := false
		for _, endpoint := range s.endpoints {
			if endpoint.command == wrapper.Command {
				isFound = true
				if err != nil && shouldLimit != nil && endpoint.isLimited {
					c.Send <- makeErrorMessage(shouldLimit, -1, wrapper.Command.Ressource)
					break
				}

				ctx := NewRequestContext(wrapper.Command, wrapper.ID, wrapper.Source)
				err = endpoint.handler(ctx, c.id, wrapper)
				if err != nil {
					c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID, wrapper.Command.Ressource)
				}

				break
			}
		}

		if isFound == false {
			c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID, wrapper.Command.Ressource)
		}
	}
}

func (s *Server) addEndpoint(command RESTCommand, isLimited bool, handler func(ctx context.Context, clientID int, frame messageFrame) error) {
	s.endpoints = append(s.endpoints, endpoint{
		command:   command,
		isLimited: isLimited,
		handler:   handler,
	})
}

func registerEndpoints(server *Server) {
	server.addEndpoint(RESTCommand{"typing", NotifyCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		return server.Messaging.BroadcastUserIsTyping(clientID, frame.Source, server, ctx)
	})

	server.addEndpoint(RESTCommand{"livesession/code", PatchCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		_, err := server.Messaging.LiveEditMessage(clientID, frame.Source, *frame.Payload.(*json.RawMessage), server, ctx)
		return err
	})

	server.addEndpoint(RESTCommand{"livesession/code", DeleteCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		_, err := server.Messaging.ToggleLiveSession(clientID, frame.Source, false, *frame.Payload.(*json.RawMessage), server, ctx)
		return err
	})

	server.addEndpoint(RESTCommand{"livesession/code", PostCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		_, err := server.Messaging.ToggleLiveSession(clientID, frame.Source, true, *frame.Payload.(*json.RawMessage), server, ctx)
		return err
	})

	server.addEndpoint(RESTCommand{"message", PatchCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		_, err := server.Messaging.EditMessage(clientID, frame.Source, *frame.Payload.(*json.RawMessage), server, ctx)
		return err
	})

	server.addEndpoint(RESTCommand{"message", PostCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		_, err := server.Messaging.SendMessage(frame.Source, clientID, *frame.Payload.(*json.RawMessage), server, ctx)
		return err
	})

	server.addEndpoint(RESTCommand{"message/read", NotifyCommandMethod}, true, func(ctx context.Context, clientID int, frame messageFrame) error {
		return server.Messaging.ReadMessages(clientID, *frame.Payload.(*json.RawMessage))
	})
}
