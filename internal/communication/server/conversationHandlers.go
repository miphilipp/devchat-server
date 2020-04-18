package server

import (
	//"errors"
	"net/http"
	//"fmt"
	"encoding/json"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
)

func (s *Webserver) deleteConversation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteConversation", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not parse path component id")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	err = s.conversationService.DeleteConversation(userID, conversationID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	s.socket.BroadcastToRoom(
		conversationID,
		websocket.RESTCommand{
			Ressource: "conversation",
			Method:    websocket.DeleteCommandMethod,
		},
		conversationID,
		-1,
	)
	s.socket.RemoveRoom(conversationID)
	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) postConversation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	requestBody := struct {
		Title          string `json:"title"`
		Repourl        string `json:"repoUrl"`
		InitialMembers []int  `json:"initialMembers"`
	}{Title: "-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "addConversation", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Title == "-" {
		apiErrorJSON := core.NewJSONFormatError("title missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	createdConversation, err := s.conversationService.CreateConversation(
		userID,
		requestBody.Title,
		requestBody.Repourl,
		requestBody.InitialMembers,
	)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
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
		s.socket.SendToClient(
			member, -1, 0,
			websocket.RESTCommand{
				Ressource: "invitation",
				Method:    websocket.PostCommandMethod,
			},
			invitation,
		)
	}

	s.socket.AddRoom(createdConversation.ID, userID)

	reply := struct {
		ID int `json:"id"`
	}{ID: createdConversation.ID}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) patchConversation(writer http.ResponseWriter, request *http.Request) {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchConversation", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not parse path component conversationID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	patchData := struct {
		Title   string `json:"title"`
		RepoURL string `json:"repoURL"`
	}{}

	err = json.NewDecoder(request.Body).Decode(&patchData)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchConversation", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	patchedConversation, err := s.conversationService.EditConversation(userContext, core.Conversation{
		ID:      conversationID,
		Title:   patchData.Title,
		Repourl: patchData.RepoURL,
	})
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	reply := struct {
		Title   string `json:"title"`
		RepoURL string `json:"repoURL"`
	}{
		Title:   patchedConversation.Title,
		RepoURL: patchedConversation.Repourl,
	}

	s.socket.BroadcastToRoom(
		conversationID,
		websocket.RESTCommand{
			Ressource: "conversation",
			Method:    websocket.PatchCommandMethod,
		},
		reply,
		-1,
	)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) getConversation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	conversations, err := s.conversationService.ListConversationsForUser(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(conversations)
}

func (s *Webserver) deleteUserFromConversation(writer http.ResponseWriter, request *http.Request) {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["conversationID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	if userContext == userID {
		newAdmin := 0
		newAdminStr := request.FormValue("newadmin")
		if newAdminStr != "" {
			a, err := strconv.Atoi(newAdminStr)
			if err != nil {
				level.Error(s.logger).Log("Handler", "deleteUserFromConversation", "err", err)
				apiErrorPath := core.NewPathFormatError("Could not parse newadmin")
				writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
				return
			}

			newAdmin = a
		}

		err = s.deleteSelfFromConversation(userContext, conversationID, newAdmin)
	} else {
		err = s.deleteOtherUserFromConversation(userContext, userID, conversationID)
	}

	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) getMembersOfConversation(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMembersOfConversation", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	users, err := s.conversationService.ListUsersOfConversation(userID, conversationID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(users)
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

	s.socket.BroadcastToRoom(
		conversationID,
		websocket.RESTCommand{
			Ressource: "conversation/member",
			Method:    websocket.DeleteCommandMethod,
		},
		reply,
		-1,
	)

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

	s.socket.BroadcastToRoom(
		conversationID,
		websocket.RESTCommand{
			Ressource: "conversation/member",
			Method:    websocket.DeleteCommandMethod,
		},
		reply,
		-1,
	)
	return nil
}

func (s *Webserver) patchAdminStatus(writer http.ResponseWriter, request *http.Request) {
	userContext := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["conversationID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	requestBody := struct {
		State bool `json:"state"`
	}{}
	err = json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchAdminStatus", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	err = s.conversationService.SetAdminStatus(userContext, userID, conversationID, requestBody.State)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	reply := struct {
		UserID int  `json:"userId"`
		State  bool `json:"state"`
	}{userID, requestBody.State}

	s.socket.BroadcastToRoom(
		conversationID,
		websocket.RESTCommand{
			Ressource: "conversation/member",
			Method:    websocket.PatchCommandMethod,
		},
		reply,
		-1,
	)

	writer.WriteHeader(http.StatusOK)
}
