package server

import (
	//"errors"

	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log/level"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
)

func (s *Webserver) postInvitation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)

	var invitation core.Invitation
	err := json.NewDecoder(request.Body).Decode(&invitation)
	if err != nil {
		level.Error(s.logger).Log("Handler", "postInvitation", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	err = s.conversationService.InviteUser(userID, invitation.Recipient, invitation.ConversationID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	command := websocket.RESTCommand{
		Ressource: "invitation",
		Method:    websocket.PostCommandMethod,
	}

	ctx := websocket.NewRequestContext(command, -1, invitation.ConversationID)
	s.socket.Unicast(ctx, invitation.Recipient, invitation)

	user, err := s.userService.GetUserForID(invitation.Recipient)
	if err != nil {
		broadcast := struct {
			core.Invitation
			RecipientName string `json:"recipientName"`
		}{
			core.Invitation{
				ConversationID:    invitation.ConversationID,
				ConversationTitle: invitation.ConversationTitle,
				Recipient:         invitation.Recipient,
			},
			user.Name,
		}

		ctx := websocket.NewRequestContext(command, -1, invitation.ConversationID)
		s.socket.BroadcastToRoom(invitation.ConversationID, broadcast, ctx)
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) deleteInvitation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	var invitation core.Invitation
	err := json.NewDecoder(request.Body).Decode(&invitation)
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteInvitation", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	err = s.conversationService.RevokeInvitation(userID, invitation.Recipient, invitation.ConversationID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "invitation",
		Method:    websocket.DeleteCommandMethod,
	}, -1, invitation.ConversationID)
	s.socket.BroadcastToRoom(invitation.ConversationID, invitation, ctx)

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) patchInvitation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	requestBody := struct {
		Action         string `json:"action"`
		ConversationID int    `json:"conversationId"`
	}{Action: "-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchInvitation", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Action == "-" {
		apiErrorJSON := core.NewJSONFormatError("action missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	conversationID := requestBody.ConversationID
	if requestBody.Action == "denie" {
		err = s.conversationService.DenieInvitation(userID, conversationID)
		if err != nil {
			if !checkForAPIError(err, writer) {
				writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
			}
			return
		}

		reply := struct {
			ConversationID int `json:"conversationId"`
			Recipient      int `json:"recipient"`
		}{conversationID, userID}

		ctx := websocket.NewRequestContext(websocket.RESTCommand{
			Ressource: "invitation",
			Method:    websocket.DeleteCommandMethod,
		}, -1, conversationID)
		s.socket.BroadcastToRoom(conversationID, reply, ctx)

	} else if requestBody.Action == "accept" {
		colorIndex, err := s.conversationService.JoinConversation(userID, conversationID)
		if err != nil {
			if !checkForAPIError(err, writer) {
				writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
			}
			return
		}

		reply := struct {
			ConversationID int `json:"conversationId"`
			Recipient      int `json:"recipient"`
			ColorIndex     int `json:"colorIndex"`
		}{conversationID, userID, colorIndex}

		ctx := websocket.NewRequestContext(websocket.RESTCommand{
			Ressource: "invitation",
			Method:    websocket.PatchCommandMethod,
		}, -1, conversationID)
		s.socket.BroadcastToRoom(conversationID, reply, ctx)

		s.socket.JoinRoom(conversationID, userID)
	} else {
		level.Warn(s.logger).Log("Handler", "patchInvitation", "err", "Invalid Action")
		http.Error(writer, "Invalid Action", http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) getInvitations(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	invitations, err := s.conversationService.ListInvitations(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(invitations)
}
