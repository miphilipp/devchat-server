package server

import (
	"net/http"
	//"fmt"
	"encoding/json"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
)

func (s *Webserver) getMessages(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not pares path component conversationID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	beforeInSequence := core.MaxInt
	beforeInSequenceStr := request.FormValue("before")
	if beforeInSequenceStr != "" {
		o, err := strconv.Atoi(beforeInSequenceStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			apiErrorPath := core.NewPathFormatError("Could not parse before")
			writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
			return
		}

		beforeInSequence = o
	}

	messageType := core.UndefinedMesssageType
	messageTypeStr := request.FormValue("type")
	if messageTypeStr != "" {
		t, err := strconv.Atoi(messageTypeStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			apiErrorPath := core.NewPathFormatError("Could not parse type")
			writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
			return
		}
		messageType = core.MessageType(t)
	}

	limit := 20
	limitStr := request.FormValue("limit")
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			apiErrorPath := core.NewPathFormatError("Could not parse limit")
			writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
			return
		}
		limit = l
	}

	messages, err := s.messageService.ListAllMessages(userID, conversationID, beforeInSequence, limit, messageType)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(messages)
}

func (s *Webserver) getCodeOfMessage(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not pares path component conversationID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	messageID, err := strconv.Atoi(vars["messageID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not pares path component messageID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	code, err := s.messageService.GetCodeOfMessage(userID, conversationID, messageID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	reply := struct {
		Code string `json:"code"`
	}{code}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) getProgrammingLanguages(writer http.ResponseWriter, request *http.Request) {
	languages, err := s.messageService.ListProgrammingLanguages()
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(languages)
}

func (s *Webserver) getMessage(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessage", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not pares path component conversationID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	messageID, err := strconv.Atoi(vars["messageID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessage", "err", err)
		apiErrorPath := core.NewPathFormatError("Could not pares path component messageID")
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	message, err := s.messageService.GetMessage(userID, conversationID, messageID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(message)
}

func (s *Webserver) serveMediaMessageRessource(writer http.ResponseWriter, request *http.Request) {

}
