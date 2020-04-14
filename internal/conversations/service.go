package conversations

import (
	//"errors"
	//"fmt"
	core "github.com/miphilipp/devchat-server/internal"
)

type Service interface {
	// Internal
	ListConversations() ([]core.Conversation, error)

	// conversation member only access
	LeaveConversation(userID, conversationID, newAdmin int) error
	ListUsersOfConversation(userCtx int, conversationID int) ([]core.UserInConversation, error)

	// Admin only access
	InviteUser(userCtx, recipient, conversationID int) error
	RevokeInvitation(userCtx, userID, conversationID int) error
	RemoveUserFromConversation(userCtx, userID, conversationID int) error
	DeleteConversation(userID, conversationID int) error
	SetAdminStatus(userID, newAdmin, conversationID int, status bool) error
	EditConversation(userCtx int, conversation core.Conversation) (core.Conversation, error)

	// Restricted access
	ListConversationsForUser(user int) ([]core.Conversation, error)
	ListInvitations(userID int) ([]core.Invitation, error)
	DenieInvitation(userID, conversationID int) error
	JoinConversation(userID, conversationID int) (int, error)
	CreateConversation(userID int, title, repoURL string, initialMembers []int) (core.Conversation, error)
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
		err = s.conversationRepo.RemoveGroupAssociation(userID, conversationID)
		if err != nil {
			return err
		}
	} else {
		return core.ErrAccessDenied
	}

	return nil
}

func (s *service) DenieInvitation(userID int, conversationID int) error {
	return s.conversationRepo.RemoveGroupAssociation(userID, conversationID)
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

func (s *service) DeleteConversation(userID int, conversationID int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userID, conversationID)
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

func (s *service) JoinConversation(userID int, conversationID int) (int, error) {
	return s.conversationRepo.MarkAsJoined(userID, conversationID)
}

func (s *service) ListInvitations(userID int) ([]core.Invitation, error) {
	return s.conversationRepo.FindInvitations(userID)
}

func (s *service) CreateConversation(userID int, title, repoURL string, initialMembers []int) (core.Conversation, error) {
	if title == "" {
		return core.Conversation{}, core.NewInvalidValueError("title")
	}

	conversation := core.Conversation{
		Title: title,
		Repourl: repoURL,
		ID: -1,
	}
	return s.conversationRepo.CreateConversation(userID, conversation, initialMembers)
}

func (s *service) ListConversationsForUser(user int) ([]core.Conversation, error) {
	return s.conversationRepo.FindConversationsForUser(user)
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


	return s.conversationRepo.MarkAsInvited(recipient, conversationID)
}

// MakeAdmin makes the passed user into an admin for a given conversation.
func (s *service) SetAdminStatus(userID, newAdmin, conversationID int, status bool) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userID, conversationID)
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
func (s *service) LeaveConversation(userID, conversationID, newAdmin int) error {
	isAdmin, err := s.conversationRepo.IsUserAdminOfConveration(userID, conversationID)
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
		err = s.SetAdminStatus(userID, newAdmin, conversationID, true)
		if err != nil {
			return err
		}
	}

	return s.conversationRepo.SetAsLeft(userID, conversationID)
}
