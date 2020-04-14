package server

import (
	//"errors"
	"net/http"
	"encoding/json"

	"github.com/go-kit/kit/log/level"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
	core "github.com/miphilipp/devchat-server/internal"
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
		Method: websocket.PostCommandMethod,
	}
	s.socket.SendToClient(invitation.Recipient, -1, 0, command, invitation)

	user, err := s.userService.GetUserForID(invitation.Recipient)
	if err != nil {
		broadcast := struct {
			core.Invitation
			RecipientName string `json:"recipientName"`
		}{ 
			core.Invitation{invitation.ConversationID, invitation.ConversationTitle, invitation.Recipient},  
			user.Name,
		}
		s.socket.BroadcastToRoom(invitation.ConversationID, command, broadcast, -1)
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

	s.socket.BroadcastToRoom(
		invitation.ConversationID,
		websocket.RESTCommand{
			Ressource: "invitation",
			Method: websocket.DeleteCommandMethod,
		}, invitation, -1,
	)
	
	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) patchInvitation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	requestBody := struct{
	 	Action 			string `json:"action"`
        ConversationID  int	   `json:"conversationId"`
	}{ Action: "-" }

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

		reply := struct{
			ConversationID  int	   `json:"conversationId"`
			Recipient		int	   `json:"recipient"`
		}{ conversationID, userID }

		s.socket.BroadcastToRoom(
			conversationID,
			websocket.RESTCommand{
				Ressource: "invitation",
				Method: websocket.DeleteCommandMethod,
			}, reply, -1,
		)

	} else if requestBody.Action == "accept" {
		colorIndex, err := s.conversationService.JoinConversation(userID, conversationID)
		if err != nil {
			if !checkForAPIError(err, writer) {
				writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
			} 
			return
		}

		reply := struct{
			ConversationID  int	   `json:"conversationId"`
			Recipient		int	   `json:"recipient"`
			ColorIndex		int	   `json:"colorIndex"`
		}{ conversationID, userID, colorIndex }

		s.socket.BroadcastToRoom(
			conversationID, 
			websocket.RESTCommand{
				Ressource: "invitation",
				Method: websocket.PatchCommandMethod,
			}, reply, -1,
		)
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