package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	core "github.com/miphilipp/devchat-server/internal"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (s *service) ToggleLiveSession(
	userCtx, conversationID int,
	state bool,
	msg json.RawMessage,
	pusher core.Pusher,
	ctx context.Context) (int, error) {

	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return 0, err
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

	var reply interface{}
	if state == false {
		reply = struct {
			MessageID int `json:"messageId"`
			NewOwner  int `json:"newOwner"`
		}{stub.ID, 0}
		err = s.messageRepo.SetLockedSateForCodeMessage(stub.ID, 0)
	} else if message.LockedBy == 0 {
		reply = struct {
			MessageID int `json:"messageId"`
			NewOwner  int `json:"newOwner"`
		}{stub.ID, userCtx}
		err = s.messageRepo.SetLockedSateForCodeMessage(stub.ID, userCtx)
	}

	if err == nil {
		pusher.BroadcastToRoom(conversationID, reply, ctx)
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

	err = s.errorIFIsNotInConversation(userID, payload.ConversationID)
	if err != nil {
		return err
	}

	return s.messageRepo.SetReadFlags(userID, payload.ConversationID)
}

func (s *service) SendMessage(
	target, userID int,
	message json.RawMessage,
	pusher core.Pusher,
	ctx context.Context) (interface{}, error) {

	err := s.errorIFIsNotInConversation(userID, target)
	if err != nil {
		return nil, err
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
		pusher.BroadcastToRoom(target, answer, ctx)
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
		pusher.BroadcastToRoom(target, answer, ctx)
	case core.MediaMessageType:
		var actualMessage core.MediaMessage
		err := json.Unmarshal(message, &actualMessage)
		if err != nil {
			return nil, core.NewJSONFormatError(err.Error())
		}
		messageID, err := s.messageRepo.StoreMediaMessage(target, userID, actualMessage)
		if err != nil {
			return nil, err
		}
		actualMessage.ID = messageID
		answer = actualMessage
		pusher.Unicast(ctx, userID, answer)
	default:
		return nil, core.ErrMessageTypeNotImplemented
	}

	return answer, nil
}

func (s *service) EditMessage(
	userCtx, conversationID int,
	message json.RawMessage,
	pusher core.Pusher,
	ctx context.Context) (int, error) {

	id, err := s.editMessage(userCtx, conversationID, message, pusher, ctx)
	if err != nil {
		return 0, err
	}

	payload := struct {
		MessageID int `json:"messageId"`
	}{id}
	pusher.BroadcastToRoom(conversationID, payload, ctx)

	return id, nil
}

func (s *service) LiveEditMessage(
	userCtx, conversationID int,
	message json.RawMessage,
	pusher core.Pusher,
	ctx context.Context) (int, error) {

	id, err := s.editMessage(userCtx, conversationID, message, pusher, ctx)
	if err != nil {
		return 0, err
	}

	pusher.BroadcastToRoom(conversationID, message, ctx)

	return id, nil
}

func (s *service) AddFileToMessage(
	userCtx, conversationID, messageID int,
	fileBuffer []byte,
	pathPrefix, fileName, fileType string) (err error) {

	err = s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return err
	}

	messageFromDB, err := s.messageRepo.FindMessageStubForConversation(conversationID, messageID)
	if err != nil {
		return err
	}

	if messageFromDB.Type != core.MediaMessageType {
		return core.ErrInvalidMessageType
	}

	mediaObjID, err := s.messageRepo.CreateMediaObject(messageFromDB.ID, fileName, fileType)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fileType, "image/") {
		uniqueThumbnailFileName := fmt.Sprintf("%d-thumbnail-%s", mediaObjID, fileName)
		path := path.Join(pathPrefix, uniqueThumbnailFileName)
		thumbailFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer thumbailFile.Close()

		reader := bytes.NewReader(fileBuffer)
		err, size := resizeImage(reader, thumbailFile, 400, 400)
		if err != nil {
			os.Remove(path)
			return err
		}
		s.messageRepo.SetMetaOfMediaMessage(mediaObjID, size)
		if err != nil {
			return err
		}
	}

	if !strings.HasPrefix(fileType, "image/") &&
		!strings.HasPrefix(fileType, "audio/") &&
		!strings.HasPrefix(fileType, "video/") {
		meta := struct {
			Size int `json:"size"`
		}{Size: len(fileBuffer)}

		s.messageRepo.SetMetaOfMediaMessage(mediaObjID, meta)
		if err != nil {
			return err
		}
	}

	uniqueFileName := fmt.Sprintf("%d-%s", mediaObjID, fileName)
	err = ioutil.WriteFile(path.Join(pathPrefix, uniqueFileName), fileBuffer, 0644)
	if err != nil {
		return err
	}

	return nil
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

func (s *service) editMessage(
	userCtx, conversationID int,
	message json.RawMessage,
	pusher core.Pusher,
	ctx context.Context) (int, error) {

	err := s.errorIFIsNotInConversation(userCtx, conversationID)
	if err != nil {
		return 0, err
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

func (s *service) CompleteMessage(id int, err error) error {
	if err != nil {
		s.messageRepo.DeleteMessage(id)
		return err
	}

	dbErr := s.messageRepo.UpdateCompleteFlag(id)
	if dbErr != nil {
		return dbErr
	}
	return err
}
