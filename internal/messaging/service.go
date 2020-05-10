package messaging

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"strings"

	core "github.com/miphilipp/devchat-server/internal"
)

type Service interface {
	ListAllMessages(userID, conversationID, beforeInSequence, limit int, mType core.MessageType) ([]interface{}, error)
	ListProgrammingLanguages() ([]core.ProgrammingLanguage, error)
	GetMediaObject(userCtx, conversationID int, fileName, pathPrefix string) (core.MediaObject, *os.File, error)
	GetMessage(userCtx, conversationID, messageID int) (interface{}, error)
	GetCodeOfMessage(userCtx, conversationID, messageID int) (string, error)
	BroadcastUserIsTyping(userCtx, conversationID int, pusher core.Pusher, ctx context.Context) error

	// Mutations
	SendMessage(target, userID int, message json.RawMessage, pusher core.Pusher, ctx context.Context) (interface{}, error)
	ReadMessages(userID int, message json.RawMessage) error
	EditMessage(userCtx, conversationID int, message json.RawMessage, pusher core.Pusher, ctx context.Context) (int, error)
	LiveEditMessage(userCtx, conversationID int, message json.RawMessage, pusher core.Pusher, ctx context.Context) (int, error)
	ToggleLiveSession(userCtx, conversationID int, state bool, message json.RawMessage, pusher core.Pusher, ctx context.Context) (int, error)
	CompleteMessage(id int, err error) error

	// AddFileToMessage adds a media object to a media message.
	AddFileToMessage(userCtx, conversationID, messageID int, fileBuffer []byte, pathPrefix, fileName, fileType string) error
}

type service struct {
	messageRepo      core.MessageRepo
	conversationRepo core.ConversationRepo
}

type messageStub struct {
	Type int `json:"type"`
	ID   int `json:"id"`
}

func NewService(messageRepo core.MessageRepo, conversationRepo core.ConversationRepo) Service {
	return &service{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
	}
}

func (s *service) BroadcastUserIsTyping(userCtx, conversationID int, pusher core.Pusher, ctx context.Context) error {
	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return err
	}

	payload := struct {
		Typist int `json:"typist"`
	}{userCtx}
	pusher.BroadcastToRoom(conversationID, payload, ctx)
	return nil
}

func (s *service) ListAllMessages(
	userID int,
	conversationID int,
	beforeInSequence int,
	limit int,
	mType core.MessageType) ([]interface{}, error) {
	err := s.errorIFIsNotInConversation(userID, conversationID)
	if err != nil {
		return nil, err
	}

	switch mType {
	case core.CodeMessageType:
		return s.messageRepo.FindCodeMessagesForConversation(conversationID, beforeInSequence, limit)
	case core.TextMessageType:
		return s.messageRepo.FindTextMessagesForConversation(conversationID, beforeInSequence, limit)
	case core.MediaMessageType:
		return s.messageRepo.FindMediaMessagesForConversation(conversationID, beforeInSequence, limit)
	case core.UndefinedMesssageType:
		return s.messageRepo.FindForConversation(conversationID, beforeInSequence, limit)
	default:
		return nil, core.ErrInvalidMessageType
	}
}

func (s *service) ListProgrammingLanguages() ([]core.ProgrammingLanguage, error) {
	return s.messageRepo.FindAllProgrammingLanguages()
}

type patchData struct {
	MessageID int    `json:"messageId"`
	Patch     string `json:"patch"`
	Title     string `json:"title"`
	Language  string `json:"language"`
}

func (s *service) GetCodeOfMessage(userCtx, conversationID, messageID int) (string, error) {
	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return "", err
	}

	message, err := s.messageRepo.FindCodeMessageForID(messageID, conversationID)
	if err != nil {
		return "", err
	}

	return message.Code, nil
}

func (s *service) GetMessage(userCtx, conversationID, messageID int) (interface{}, error) {
	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return nil, err
	}

	messageFromDB, err := s.messageRepo.FindMessageStubForConversation(conversationID, messageID)
	if err != nil {
		return 0, err
	}

	switch messageFromDB.Type {
	case core.CodeMessageType:
		return s.messageRepo.FindCodeMessageForID(messageID, conversationID)
	case core.TextMessageType:
		return s.messageRepo.FindTextMessageForID(messageID, conversationID)
	case core.MediaMessageType:
		return s.messageRepo.FindMediaMessageForID(messageID, conversationID)
	default:
		return nil, core.ErrMessageTypeNotImplemented
	}
}

func (s *service) errorIFIsNotInConversation(userCtx, conversationID int) error {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return err
	}

	if !isMember {
		return core.ErrAccessDenied
	}
	return nil
}

func (s *service) GetMediaObject(userCtx, conversationID int, fileName, pathPrefix string) (core.MediaObject, *os.File, error) {
	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return core.MediaObject{}, nil, err
	}

	components := strings.SplitN(fileName, "-", 2)
	if len(components) != 2 {
		return core.MediaObject{}, nil, core.NewPathFormatError("Invalid file name")
	}

	objID, err := strconv.Atoi(components[0])
	if err != nil {
		return core.MediaObject{}, nil, core.NewInvalidValueError("media object id")
	}

	obj, err := s.messageRepo.FindMediaObjectForID(objID, conversationID)
	if err != nil {
		return core.MediaObject{}, nil, err
	}

	file, err := os.Open(path.Join(pathPrefix, fileName))
	if err != nil {
		return core.MediaObject{}, nil, err
	}

	return obj, file, nil
}
