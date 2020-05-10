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

type Server struct {
	Messaging     messaging.Service
	Conversations conversations.Service
	User          user.Service

	logger  log.Logger
	limiter *WebsocketRateLimiter

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
			s.notifyOnlineState(conversation.ID, user, true)
			s.JoinRoom(conversation.ID, c.id)
		}

		s.User.UpdateOnlineTimestamp(c.id)

		go s.sendLoop(c)
	}

	conn.SetCloseHandler(func(code int, text string) error {
		s.cleanupAfterClient(conn, c)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return nil
	})

	c.connsLock.Lock()
	c.conns = append(c.conns, conn)
	c.connsLock.Unlock()

	go s.receiveLoop(conn, c)
	return nil
}

func (s *Server) notifyOnlineState(roomNumber int, userID int, state bool) {
	notification := struct {
		UserID   int  `json:"userId"`
		NewState bool `json:"newState"`
	}{
		UserID:   userID,
		NewState: state,
	}

	ctx := NewRequestContext(RESTCommand{
		Ressource: "member/onlinestate",
		Method:    PatchCommandMethod,
	}, -1, -1)
	s.BroadcastToRoom(roomNumber, notification, ctx)
}

func (s *Server) cleanupAfterClient(conn *websocket.Conn, client *client) {
	isCompletlyDisconnected := client.breakConnection(conn)
	if isCompletlyDisconnected {
		clients.Lock()
		delete(clients.m, client.id)
		clients.Unlock()
		s.removeClientFromRooms(client)
		client.Disconnect <- 1000 // Code is not relevant

		s.rooms.RLock()
		for _, conversation := range s.rooms.m {
			s.notifyOnlineState(conversation.ID, client.id, false)
		}
		s.rooms.RUnlock()
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
			closeMsg, ok := err.(*websocket.CloseError)
			if !ok {
				level.Error(s.logger).Log("err", err)
			} else {
				if closeMsg.Code == websocket.CloseAbnormalClosure {
					s.cleanupAfterClient(conn, c)
				}
				level.Info(s.logger).Log("message", "Connection has been closed by peer")
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

		if err != nil && shouldLimit != nil && wrapper.Command.Ressource != "livecoding" {
			level.Info(s.logger).Log(
				"Event", "RateLimit",
				"RemoteAddr", conn.RemoteAddr(),
				"commandRessource", wrapper.Command.Ressource)
			c.Send <- makeErrorMessage(shouldLimit, -1)
		}

		ctx := NewRequestContext(wrapper.Command, wrapper.ID, wrapper.Source)

		switch wrapper.Command.Ressource {
		case "message":
			if wrapper.Command.Method == PostCommandMethod {
				_, err = s.Messaging.SendMessage(wrapper.Source, c.id, msg, s, ctx)
				if err != nil {
					c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
				}
			} else if wrapper.Command.Method == PatchCommandMethod {
				_, err := s.Messaging.EditMessage(c.id, wrapper.Source, msg, s, ctx)
				if err != nil {
					c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
				}
			} else {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
			}
		case "message/read":
			if wrapper.Command.Method != NotifyCommandMethod {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
				break
			}

			err = s.Messaging.ReadMessages(c.id, msg)
			if err != nil {
				c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
			}
		case "livesession/code/start":
			if wrapper.Command.Method != NotifyCommandMethod {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
				break
			}

			_, err := s.Messaging.ToggleLiveSession(c.id, wrapper.Source, true, msg, s, ctx)
			if err != nil {
				c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
			}
		case "livesession/code/stop":
			if wrapper.Command.Method != NotifyCommandMethod {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
				break
			}

			_, err := s.Messaging.ToggleLiveSession(c.id, wrapper.Source, false, msg, s, ctx)
			if err != nil {
				c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
			}
		case "livecoding":
			if wrapper.Command.Method != PatchCommandMethod {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
				break
			}

			_, err := s.Messaging.LiveEditMessage(c.id, wrapper.Source, msg, s, ctx)
			if err != nil {
				c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
			}
		case "typing":
			if wrapper.Command.Method != NotifyCommandMethod {
				c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
				break
			}

			err := s.Messaging.BroadcastUserIsTyping(c.id, wrapper.Source, s, ctx)
			if err != nil {
				c.Send <- makeErrorMessage(core.UnwrapDatabaseError(err), wrapper.ID)
			}
		default:
			c.Send <- makeErrorMessage(core.ErrUnsupportedMethod, wrapper.ID)
		}
	}
}

type endpoint struct {
	command   RESTCommand
	isLimited bool
	action    func(ctx context.Context, clientID int, frame messageFrame)
}
