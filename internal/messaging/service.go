package messaging

import (
	//"fmt"
	"encoding/json"

	core "github.com/miphilipp/devchat-server/internal"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Service interface {
	ListAllMessages(userID, conversationID, beforeInSequence, limit int, mType core.MessageType) ([]interface{}, error)
	GetMessage(userCtx, conversationID, messageID int) (interface{}, error)
	SendMessage(target, userID int, message json.RawMessage) (interface{}, error)
	ReadMessages(userID int, message json.RawMessage) error
	ListProgrammingLanguages() ([]core.ProgrammingLanguage, error)
	EditMessage(userCtx, conversationID int, message json.RawMessage) (int, error)
	GetCodeOfMessage(userCtx, conversationID, messageID int) (string, error)
	ToggleLiveSession(userCtx, conversationID int, state bool, message json.RawMessage) (int, error)
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

func (s *service) ToggleLiveSession(userCtx, conversationID int, state bool, msg json.RawMessage) (int, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return 0, err
	}
	if !isMember {
		return 0, core.ErrAccessDenied
	}

	var stub messageStub
	err = json.Unmarshal(msg, &stub)
	if err != nil {
		return 0, core.NewJSONFormatError(err.Error())
	}

	if core.MessageType(stub.Type) != core.CodeMessageType {
		return 0, core.ErrInvalidMessageType
	}

	message, err := s.messageRepo.FindCodeMessageForID(stub.ID, conversationID)
	if err != nil {
		return 0, err
	}

	if state == false {
		err = s.messageRepo.SetLockedSateForCodeMessage(stub.ID, 0)
	} else if message.LockedBy == 0 {
		err = s.messageRepo.SetLockedSateForCodeMessage(stub.ID, userCtx)
	}

	return message.ID, err
}

func (s *service) ReadMessages(userID int, message json.RawMessage) error {
	payload := struct {
		ConversationID int `json:"conversationId"`
	}{}
	err := json.Unmarshal(message, &payload)
	if err != nil {
		return core.NewJSONFormatError(err.Error())
	}

	isMember, err := s.conversationRepo.IsUserInConversation(userID, payload.ConversationID)
	if err != nil {
		return err
	}
	if !isMember {
		return core.ErrAccessDenied
	}

	return s.messageRepo.SetReadFlags(userID, payload.ConversationID)
}

func (s *service) SendMessage(target, userID int, message json.RawMessage) (interface{}, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userID, target)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, core.ErrAccessDenied
	}

	var stub messageStub
	err = json.Unmarshal(message, &stub)
	if err != nil {
		return nil, core.NewJSONFormatError(err.Error())
	}

	messageType := core.MessageType(stub.Type)
	var answer interface{}
	switch messageType {
	case core.TextMessageType:
		var actualMessage core.TextMessage
		err := json.Unmarshal(message, &actualMessage)
		if err != nil {
			return nil, core.NewJSONFormatError(err.Error())
		}
		messageID, err := s.messageRepo.StoreTextMessage(target, userID, actualMessage)
		if err != nil {
			return nil, err
		}
		actualMessage.ID = messageID
		answer = actualMessage
	case core.CodeMessageType:
		var actualMessage core.CodeMessage
		err := json.Unmarshal(message, &actualMessage)
		if err != nil {
			return nil, core.NewJSONFormatError(err.Error())
		}
		messageID, err := s.messageRepo.StoreCodeMessage(target, userID, actualMessage)
		if err != nil {
			return nil, err
		}
		actualMessage.ID = messageID
		answer = actualMessage
	default:
		return nil, core.ErrMessageTypeNotImplemented
	}

	return answer, nil
}

func (s *service) ListAllMessages(
	userID int,
	conversationID int,
	beforeInSequence int,
	limit int,
	mType core.MessageType) ([]interface{}, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userID, conversationID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, core.ErrAccessDenied
	}

	switch mType {
	case core.CodeMessageType:
		return s.messageRepo.FindCodeMessagesForConversation(conversationID, beforeInSequence, limit)
	case core.TextMessageType:
		return s.messageRepo.FindTextMessagesForConversation(conversationID, beforeInSequence, limit)
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

func (s *service) applyPatchDataToCodeMessage(userCtx, conversationID int, patchData patchData) error {
	codeMessage, err := s.messageRepo.FindCodeMessageForID(patchData.MessageID, conversationID)
	if err != nil {
		return err
	}

	if codeMessage.LockedBy > 0 && codeMessage.LockedBy != userCtx {
		return core.ErrAccessDenied
	}

	updatedCode := codeMessage.Code
	if patchData.Patch != "" {
		dmp := diffmatchpatch.New()
		patches, err := dmp.PatchFromText(patchData.Patch)
		if err != nil {
			return err
		}
		updatedCode, _ = dmp.PatchApply(patches, codeMessage.Code)
	}

	updatedTitle := codeMessage.Title
	if patchData.Title != "" {
		updatedTitle = patchData.Title
	}

	updatedLanguage := codeMessage.Language
	if patchData.Language != "" {
		updatedLanguage = patchData.Language
	}

	err = s.messageRepo.UpdateCode(patchData.MessageID, updatedCode, updatedTitle, updatedLanguage)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) EditMessage(userCtx, conversationID int, message json.RawMessage) (int, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return 0, err
	}
	if !isMember {
		return 0, core.ErrAccessDenied
	}

	var patchInfo = struct {
		ID int `json:"messageId"`
	}{}
	err = json.Unmarshal(message, &patchInfo)
	if err != nil {
		return 0, core.NewJSONFormatError(err.Error())
	}

	messageFromDB, err := s.messageRepo.FindMessageStubForConversation(conversationID, patchInfo.ID)
	if err != nil {
		return 0, err
	}

	switch messageFromDB.Type {
	case core.CodeMessageType:
		var concretePath patchData
		err = json.Unmarshal(message, &concretePath)
		if err != nil {
			return 0, core.NewJSONFormatError(err.Error())
		}

		err = s.applyPatchDataToCodeMessage(userCtx, conversationID, concretePath)
		if err != nil {
			return 0, err
		}
	default:
		return 0, core.ErrMessageTypeNotImplemented
	}

	return patchInfo.ID, nil
}

func (s *service) GetCodeOfMessage(userCtx, conversationID, messageID int) (string, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return "", err
	}

	if !isMember {
		return "", core.ErrAccessDenied
	}

	message, err := s.messageRepo.FindCodeMessageForID(messageID, conversationID)
	if err != nil {
		return "", err
	}

	return message.Code, nil
}

func (s *service) GetMessage(userCtx, conversationID, messageID int) (interface{}, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, core.ErrAccessDenied
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
	default:
		return nil, core.ErrMessageTypeNotImplemented
	}
}
