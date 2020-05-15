package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
)

func (s *Webserver) deleteConversation(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteConversation", "err", err)
		return core.NewPathFormatError("Could not parse path component id")
	}

	err = s.conversationService.DeleteConversation(userID, conversationID)
	if err != nil {
		return err
	}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "conversation",
		Method:    websocket.DeleteCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, conversationID, ctx)

	s.socket.RemoveRoom(conversationID)
	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) postConversation(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	requestBody := struct {
		Title          string `json:"title"`
		Repourl        string `json:"repoUrl"`
		InitialMembers []int  `json:"initialMembers"`
	}{Title: "-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "addConversation", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.Title == "-" {
		return core.NewJSONFormatError("title missing")
	}

	createdConversation, err := s.conversationService.CreateConversation(
		userID,
		requestBody.Title,
		requestBody.Repourl,
		requestBody.InitialMembers,
	)
	if err != nil {
		return err
	}

	for _, member := range requestBody.InitialMembers {
		if member == userID {
			continue
		}
		invitation := core.Invitation{
			ConversationID:    createdConversation.ID,
			ConversationTitle: createdConversation.Title,
			Recipient:         member,
		}

		ctx := websocket.NewRequestContext(websocket.RESTCommand{
			Ressource: "invitation",
			Method:    websocket.PostCommandMethod,
		}, -1, userID)
		s.socket.Unicast(ctx, member, invitation)
	}

	s.socket.AddRoom(createdConversation.ID, userID)

	reply := struct {
		ID int `json:"id"`
	}{ID: createdConversation.ID}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
	return nil
}

func (s *Webserver) patchConversation(writer http.ResponseWriter, request *http.Request) error {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchConversation", "err", err)
		return core.NewPathFormatError("Could not parse path component conversationID")
	}

	patchData := struct {
		Title   string `json:"title"`
		RepoURL string `json:"repoURL"`
	}{}

	err = json.NewDecoder(request.Body).Decode(&patchData)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchConversation", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	patchedConversation, err := s.conversationService.EditConversation(userContext, core.Conversation{
		ID:      conversationID,
		Title:   patchData.Title,
		Repourl: patchData.RepoURL,
	})
	if err != nil {
		return err
	}

	reply := struct {
		Title   string `json:"title"`
		RepoURL string `json:"repoURL"`
	}{
		Title:   patchedConversation.Title,
		RepoURL: patchedConversation.Repourl,
	}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "conversation",
		Method:    websocket.PatchCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, reply, ctx)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
	return nil
}

func (s *Webserver) getConversation(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	conversations, err := s.conversationService.ListConversationsForUser(userID)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(conversations)
	return nil
}

func (s *Webserver) deleteUserFromConversation(writer http.ResponseWriter, request *http.Request) error {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["conversationID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	if userContext == userID {
		newAdmin := 0
		newAdminStr := request.FormValue("newadmin")
		if newAdminStr != "" {
			a, err := strconv.Atoi(newAdminStr)
			if err != nil {
				level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
				return core.NewPathFormatError("Could not parse newadmin")
			}

			newAdmin = a
		}

		err = s.deleteSelfFromConversation(userContext, conversationID, newAdmin)
	} else {
		err = s.deleteOtherUserFromConversation(userContext, userID, conversationID)
	}

	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) getMembersOfConversation(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMembersOfConversation", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	users, err := s.conversationService.ListUsersOfConversation(userID, conversationID)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(users)
	return nil
}

func (s *Webserver) deleteSelfFromConversation(userID, conversationID, newAdmin int) error {
	err := s.conversationService.LeaveConversation(userID, conversationID, newAdmin)
	if err != nil {
		return err
	}

	s.socket.RemoveClientFromRoom(conversationID, userID)

	reply := struct {
		UserID         int `json:"userId"`
		ConversationID int `json:"conversationId"`
		NewAdminID     int `json:"newAdminId"`
	}{userID, conversationID, newAdmin}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "conversation/member",
		Method:    websocket.DeleteCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, reply, ctx)

	return nil
}

func (s *Webserver) deleteOtherUserFromConversation(userCtx, userID, conversationID int) error {
	err := s.conversationService.RemoveUserFromConversation(userCtx, userID, conversationID)
	if err != nil {
		return err
	}

	s.socket.RemoveClientFromRoom(conversationID, userID)

	reply := struct {
		UserID         int `json:"userId"`
		ConversationID int `json:"conversationId"`
	}{userID, conversationID}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "conversation/member",
		Method:    websocket.DeleteCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, reply, ctx)
	return nil
}

func (s *Webserver) patchAdminStatus(writer http.ResponseWriter, request *http.Request) error {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["conversationID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	requestBody := struct {
		State bool `json:"state"`
	}{}
	err = json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	err = s.conversationService.SetAdminStatus(userContext, userID, conversationID, requestBody.State)
	if err != nil {
		return err
	}

	reply := struct {
		UserID int  `json:"userId"`
		State  bool `json:"state"`
	}{userID, requestBody.State}

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "conversation/member",
		Method:    websocket.PatchCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, reply, ctx)

	writer.WriteHeader(http.StatusOK)
	return nil
}
