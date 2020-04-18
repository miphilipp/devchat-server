package messaging

import (
	"encoding/json"
	"time"

	"github.com/go-kit/kit/log"
	core "github.com/miphilipp/devchat-server/internal"
)

type loggingService struct {
	logger  log.Logger
	next    Service
	verbose bool
}

func NewLoggingService(logger log.Logger, s Service, verbose bool) Service {
	return &loggingService{logger, s, verbose}
}

func (s *loggingService) ReadMessages(userID int, message json.RawMessage) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ReadMessages",
				"userID", userID,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ReadMessages(userID, message)
}

func (s *loggingService) SendMessage(target int, userID int, message json.RawMessage) (answer interface{}, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "SendMessage",
				"userID", userID,
				"target", target,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.SendMessage(target, userID, message)
}

func (s *loggingService) ListAllMessages(userID int, conversationID int, beforeInSequence int, limit int, mType core.MessageType) (messages []interface{}, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListAllMessages",
				"userID", userID,
				"conversationID", conversationID,
				"beforeInSequence", beforeInSequence,
				"type", mType,
				"limit", limit,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ListAllMessages(userID, conversationID, beforeInSequence, limit, mType)
}

func (s *loggingService) ListProgrammingLanguages() (languages []core.ProgrammingLanguage, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListProgrammingLanguages",
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ListProgrammingLanguages()
}

func (s *loggingService) GetCodeOfMessage(userCtx int, conversationID int, messageID int) (code string, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "GetCodeOfMessage",
				"userCtx", userCtx,
				"messageID", messageID,
				"conversationID", conversationID,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.GetCodeOfMessage(userCtx, conversationID, messageID)
}

func (s *loggingService) EditMessage(userCtx int, conversationID int, message json.RawMessage) (messageID int, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "EditMessage",
				"userCtx", userCtx,
				"conversationID", conversationID,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.EditMessage(userCtx, conversationID, message)
}

func (s *loggingService) ToggleLiveSession(userCtx, conversationID int, state bool, message json.RawMessage) (messageID int, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ToggleLiveSession",
				"userCtx", userCtx,
				"conversationID", conversationID,
				"state", state,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ToggleLiveSession(userCtx, conversationID, state, message)
}

func (s *loggingService) GetMessage(userCtx, conversationID, messageID int) (message interface{}, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "GetMessage",
				"userCtx", userCtx,
				"conversationID", conversationID,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.GetMessage(userCtx, conversationID, messageID)
}
