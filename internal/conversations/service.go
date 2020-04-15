package conversations

import (
	//"fmt"
	core "github.com/miphilipp/devchat-server/internal"
)

// Service defines all use cases related to conversations.
// All users that are passed via an argument labeled userCtx are expected to be logged in.
type Service interface {
	// Internal
	ListConversations() ([]core.Conversation, error)

	// conversation member only access
	LeaveConversation(userCtx, conversationID, newAdmin int) error
	ListUsersOfConversation(userCtx int, conversationID int) ([]core.UserInConversation, error)

	// Admin only access
	InviteUser(userCtx, recipient, conversationID int) error
	RevokeInvitation(userCtx, userID, conversationID int) error
	RemoveUserFromConversation(userCtx, userID, conversationID int) error
	DeleteConversation(userCtx, conversationID int) error
	SetAdminStatus(userCtx, newAdmin, conversationID int, status bool) error
	EditConversation(userCtx int, conversation core.Conversation) (core.Conversation, error)

	// Restricted access
	ListConversationsForUser(userCtx int) ([]core.Conversation, error)
	ListInvitations(userCtx int) ([]core.Invitation, error)
	DenieInvitation(userCtx, conversationID int) error
	JoinConversation(userCtx, conversationID int) (int, error)
	CreateConversation(userCtx int, title, repoURL string, initialMembers []int) (core.Conversation, error)
}

type service struct {
	conversationRepo core.ConversationRepo
}

// NewService creates and returns new Service
func NewService(conversationRepo core.ConversationRepo) Service  {
	return &service{
		conversationRepo: conversationRepo,
	}
}

func (s *service) RevokeInvitation(userCtx, userID, conversationID int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	if userID == 0 {
		return core.NewInvalidValueError("userId")
	}

	if isAdmin {
		isMember, err := s.conversationRepo.IsUserInConversation(userID, conversationID)
		if err != nil {
			return err
		}

		if isMember {
			return core.ErrRessourceDoesNotExist
		}

		err = s.conversationRepo.RemoveGroupAssociation(userID, conversationID)
		if err != nil {
			return err
		}
	} else {
		return core.ErrAccessDenied
	}

	return nil
}

func (s *service) DenieInvitation(userCtx int, conversationID int) error {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return err
	}

	if isMember {
		return core.ErrRessourceDoesNotExist
	}

	return s.conversationRepo.RemoveGroupAssociation(userCtx, conversationID)
}

func (s *service) RemoveUserFromConversation(userCtx, userID, conversationID int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	if isAdmin {
		err = s.conversationRepo.RemoveGroupAssociation(userID, conversationID)
		if err != nil {
			return err
		}
	} else {
		return core.ErrAccessDenied
	}

	return nil
}

func (s *service) DeleteConversation(userCtx int, conversationID int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	if isAdmin {
		err = s.conversationRepo.DeleteConversation(conversationID)
		if err != nil {
			return err
		}
	} else {
		return core.ErrAccessDenied
	}

	return nil
}

func (s *service) JoinConversation(userCtx int, conversationID int) (int, error) {
	return s.conversationRepo.MarkAsJoined(userCtx, conversationID)
}

func (s *service) ListInvitations(userCtx int) ([]core.Invitation, error) {
	return s.conversationRepo.FindInvitations(userCtx)
}

func (s *service) CreateConversation(userCtx int, title, repoURL string, initialMembers []int) (core.Conversation, error) {
	if title == "" {
		return core.Conversation{}, core.NewInvalidValueError("title")
	}

	conversation := core.Conversation{
		Title: title,
		Repourl: repoURL,
		ID: -1,
	}
	return s.conversationRepo.CreateConversation(userCtx, conversation, initialMembers)
}

func (s *service) ListConversationsForUser(userCtx int) ([]core.Conversation, error) {
	return s.conversationRepo.FindConversationsForUser(userCtx)
}

func (s *service) ListConversations() ([]core.Conversation, error) {
	return s.conversationRepo.FindConversations()
}

// InviteUser adds an unaccepted inviation to an user.
func (s *service) InviteUser(userCtx, recipient, conversationID int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	if !isAdmin {
		return core.ErrAccessDenied
	}

	isMember, err := s.conversationRepo.IsUserInConversation(recipient, conversationID)
	if err != nil {
		return err
	}

	if isMember {
		return core.ErrAlreadyExists
	}

	return s.conversationRepo.MarkAsInvited(recipient, conversationID)
}

// MakeAdmin makes the passed user into an admin for a given conversation.
func (s *service) SetAdminStatus(userCtx, newAdmin, conversationID int, status bool) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	if !isAdmin {
		return core.ErrAccessDenied
	}

	isMember, err := s.conversationRepo.IsUserInConversation(newAdmin, conversationID)
	if err != nil {
		return err
	}

	if !isMember {
		return core.ErrUserDoesNotExist
	}

	return s.conversationRepo.SetAdminState(newAdmin, conversationID, status)
}

func (s *service) EditConversation(userCtx int, conversation core.Conversation) (core.Conversation, error) {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversation.ID)
	if err != nil {
		return core.Conversation{}, err
	}

	if !isAdmin {
		return core.Conversation{}, core.ErrAccessDenied
	}

	currentValues, err := s.conversationRepo.FindConversationForID(conversation.ID)
	if err != nil {
		return core.Conversation{}, err
	}

	if conversation.Title == "" {
		conversation.Title = currentValues.Title
	}
	
	err = s.conversationRepo.SetMetaDataOfConversation(conversation)
	if err != nil {
		return core.Conversation{}, err
	}

	return core.Conversation{
		ID: conversation.ID,
		Title: conversation.Title,
		Repourl: conversation.Repourl,
	}, nil
}

func (s *service) ListUsersOfConversation(userCtx int, conversationID int) ([]core.UserInConversation, error) {
	isMember, err := s.conversationRepo.IsUserInConversation(userCtx, conversationID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, core.ErrAccessDenied
	}

	users, err := s.conversationRepo.GetUsersInConversation(conversationID)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// LeaveConversation removes a user from a conversation
func (s *service) LeaveConversation(userCtx, conversationID, newAdmin int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userCtx, conversationID)
	if err != nil {
		return err
	}

	numberOfAdmins, err := s.conversationRepo.CountAdminsOfConversation(conversationID)
	if err != nil {
		return err
	}

	if isAdmin && numberOfAdmins == 1 && newAdmin == 0 {
		return core.NewInvalidValueError("newAdmin")
	}

	if isAdmin && numberOfAdmins == 1 {
		err = s.SetAdminStatus(userCtx, newAdmin, conversationID, true)
		if err != nil {
			return err
		}
	}

	return s.conversationRepo.SetAsLeft(userCtx, conversationID)
}
