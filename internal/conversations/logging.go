package conversations

import (
	"time"
	"github.com/go-kit/kit/log"
	core "github.com/miphilipp/devchat-server/internal"
)

type loggingService struct {
	logger  log.Logger
	next    Service
	verbose bool
}

func NewLoggingService(logger log.Logger, s Service, verbose bool) Service  {
	return &loggingService{logger, s, verbose}
}

func (s *loggingService) DeleteConversation(userID int, conversationID int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "DeleteConversation", 
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.DeleteConversation(userID, conversationID)
}

func (s *loggingService) JoinConversation(userID int, conversationID int) (colorIndex int, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "JoinConversation", 
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.JoinConversation(userID, conversationID)
}

func (s *loggingService) ListInvitations(userID int) (inviations []core.Invitation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListInvitations", 
				"userID", userID, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.ListInvitations(userID)
}

func (s *loggingService) CreateConversation(userID int, title, repoURL string, initialContacts []int) (res core.Conversation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "CreateConversation", 
				"userID", userID, 
				"title", title,
				"repoURL", repoURL,
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.CreateConversation(userID, title, repoURL, initialContacts)
}

func (s *loggingService) EditConversation(userCtx int, conversation core.Conversation) (c core.Conversation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "EditConversation", 
				"userCtx", userCtx, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.EditConversation(userCtx, conversation)
}

func (s *loggingService) ListConversationsForUser(user int) (conversations []core.Conversation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListConversationsForUser", 
				"userID", user, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.ListConversationsForUser(user)
}

func (s *loggingService) ListConversations() (conversations []core.Conversation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListConversations", 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.ListConversations()
}

// InviteUser adds an unaccepted inviation to an user.
func (s *loggingService) InviteUser(userCtx int, userID int, conversation int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "InviteUser", 
				"userCtx", userCtx,
				"userID", userID, 
				"conversationID", conversation, 
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.InviteUser(userCtx, userID, conversation)
}

func (s *loggingService) RevokeInvitation(userCtx, userID, conversationID int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "RevokeInvitation", 
				"userCtx", userCtx,
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.RevokeInvitation(userCtx, userID, conversationID)
}

func (s *loggingService) DenieInvitation(userID int, conversationID int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "DenieInvitation", 
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.DenieInvitation(userID, conversationID)
}

// MakeAdmin makes the passed user into an admin for a given conversation
func (s *loggingService) SetAdminStatus(userID, newAdmin, conversationID int, status bool) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "SetAdminStatus", 
				"userID", userID, 
				"newAdminID", newAdmin, 
				"conversationID", conversationID, 
				"status", status,
				"took", time.Since(begin), 
				"err", err)
		}
	}(time.Now())
	return s.next.SetAdminStatus(userID, newAdmin, conversationID, status)
}

func (s *loggingService) RemoveUserFromConversation(userCtx, userID, conversationID int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "RemoveUserFromConversation", 
				"userContext", userCtx,
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.RemoveUserFromConversation(userCtx, userID, conversationID)
}

func (s *loggingService) ListUsersOfConversation(userCtx int, conversationID int) (cs []core.UserInConversation, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ListUsersOfConversation", 
				"userCtx", userCtx, 
				"conversationID", conversationID, 
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ListUsersOfConversation(userCtx, conversationID)
}

// LeaveConversation removes a user from a conversation
func (s *loggingService) LeaveConversation(userID, conversationID, newAdmin int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "LeaveConversation", 
				"userID", userID, 
				"conversationID", conversationID, 
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.LeaveConversation(userID, conversationID, newAdmin)
}
