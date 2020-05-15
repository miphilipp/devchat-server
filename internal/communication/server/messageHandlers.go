package server

import (
	"bufio"
	"net/http"
	"time"

	"encoding/json"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
)

func (s *Webserver) getMessages(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		return core.NewPathFormatError("Could not pares path component conversationID")
	}

	beforeInSequence := core.MaxInt
	beforeInSequenceStr := request.FormValue("before")
	if beforeInSequenceStr != "" {
		o, err := strconv.Atoi(beforeInSequenceStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			return core.NewPathFormatError("Could not parse before")
		}

		beforeInSequence = o
	}

	messageType := core.UndefinedMesssageType
	messageTypeStr := request.FormValue("type")
	if messageTypeStr != "" {
		t, err := strconv.Atoi(messageTypeStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			return core.NewPathFormatError("Could not parse type")
		}
		messageType = core.MessageType(t)
	}

	limit := 20
	limitStr := request.FormValue("limit")
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			level.Error(s.logger).Log("Handler", "getMessages", "err", err)
			return core.NewPathFormatError("Could not parse limit")
		}
		limit = l
	}

	messages, err := s.messageService.ListAllMessages(userID, conversationID, beforeInSequence, limit, messageType)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(messages)
	return nil
}

func (s *Webserver) getCodeOfMessage(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		return core.NewPathFormatError("Could not pares path component conversationID")
	}

	messageID, err := strconv.Atoi(vars["messageID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessages", "err", err)
		return core.NewPathFormatError("Could not pares path component messageID")
	}

	code, err := s.messageService.GetCodeOfMessage(userID, conversationID, messageID)
	if err != nil {
		return err
	}

	reply := struct {
		Code string `json:"code"`
	}{code}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
	return nil
}

func (s *Webserver) getProgrammingLanguages(writer http.ResponseWriter, request *http.Request) error {
	languages, err := s.messageService.ListProgrammingLanguages()
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(languages)
	return nil
}

func (s *Webserver) getMessage(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)
	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessage", "err", err)
		return core.NewPathFormatError("Could not pares path component conversationID")
	}

	messageID, err := strconv.Atoi(vars["messageID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "getMessage", "err", err)
		return core.NewPathFormatError("Could not pares path component messageID")
	}

	message, err := s.messageService.GetMessage(userID, conversationID, messageID)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(message)
	return nil
}

func (s *Webserver) uploadMedia(writer http.ResponseWriter, request *http.Request) error {

	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)

	conversationID, err := strconv.Atoi(vars["id"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "uploadMedia", "err", err)
		return core.NewPathFormatError("Could not parse path component conversationID")
	}

	messageID, err := strconv.Atoi(vars["messageID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "uploadMedia", "err", err)
		return core.NewPathFormatError("Could not parse path component messageID")
	}

	err = request.ParseMultipartForm(2 << 20)
	if err != nil {
		level.Error(s.logger).Log("Handler", "uploadMedia", "err", err)
		return core.ErrUnknownError
	}

	var mediaCreationErr error
	files := request.MultipartForm.File["files"]
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			level.Error(s.logger).Log("Handler", "uploadMedia", "err", err)
			mediaCreationErr = err
			break
		}
		defer file.Close()

		buf := bufio.NewReaderSize(file, int(header.Size))
		peakAmount := min(header.Size, 512)
		sniff, err := buf.Peek(int(peakAmount))
		if err != nil {
			level.Error(s.logger).Log("Handler", "uploadMedia", "err", err)
			mediaCreationErr = err
			break
		}

		contentType := http.DetectContentType(sniff)
		buffer := make([]byte, header.Size)
		buf.Read(buffer)

		err = s.messageService.AddFileToMessage(
			userID,
			conversationID,
			messageID,
			buffer,
			s.config.MediaFolder,
			header.Filename,
			contentType,
		)
		if err != nil {
			mediaCreationErr = err
			break
		}
	}

	err = s.messageService.CompleteMessage(messageID, mediaCreationErr)
	if err != nil {
		return err
	}

	message, err := s.messageService.GetMessage(userID, conversationID, messageID)
	if err != nil {
		return err
	}
	mediaMessage := message.(core.MediaMessage)

	ctx := websocket.NewRequestContext(websocket.RESTCommand{
		Ressource: "message",
		Method:    websocket.PostCommandMethod,
	}, -1, conversationID)
	s.socket.BroadcastToRoom(conversationID, mediaMessage, ctx)
	return nil
}

func (s *Webserver) serveMediaMessageRessource(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	vars := mux.Vars(request)

	conversationID, err := strconv.Atoi(vars["conversationID"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "serveMediaMessageRessource", "err", err)
		return core.NewPathFormatError("Could not parse path component conversationID")
	}

	fileName := vars["fileName"]
	mediaObj, file, err := s.messageService.GetMediaObject(
		userID,
		conversationID,
		fileName,
		s.config.MediaFolder,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	writer.Header().Set("Cache-Control", "max-age=5552000")
	writer.Header().Set("Content-Type", mediaObj.MIMEType)
	http.ServeContent(writer, request, "", time.Time{}, file)
	return nil
}

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}
