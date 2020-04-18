package core

import (
	"time"

	"github.com/google/uuid"
)

// ConversationRepo contains all queries and mutations to work with conversations.
type ConversationRepo interface {

	// Mutations
	DeleteConversation(id int) error
	CreateConversation(userID int, c Conversation, initialMembers []int) (Conversation, error)
	MarkAsInvited(userID, conversationID int) error
	MarkAsJoined(userID, conversationID int) (int, error)
	RemoveGroupAssociation(userID, conversationID int) error
	SetMetaDataOfConversation(conversation Conversation) error
	SetAsLeft(userID, conversationID int) error
	SetAdminState(userID, conversationID int, state bool) error

	// Queries
	FindInvitations(userid int) ([]Invitation, error)
	FindConversations() ([]Conversation, error)
	FindConversationForID(conversationID int) (Conversation, error)
	FindConversationsForUser(userid int) ([]Conversation, error)
	IsUserInConversation(userID, conversationID int) (bool, error)
	IsUserAdminOfConveration(userID, conversationID int) (bool, error)
	GetUsersInConversation(conversationID int) ([]UserInConversation, error)
	CountAdminsOfConversation(conversationID int) (int, error)
}

// UserRepo
type UserRepo interface {

	// Mutations
	LockUser(userID int) error
	UnlockUser(userID int) error
	SetConfirmationIDToNULL(token string) (string, error)
	SoftDeleteUser(userid int) error
	CreateUser(user User, password string) (User, error)
	IncrementFailedLoginAttempts(user string) error
	SetPassword(user int, newPassword string) error
	CreateRecoverID(emailAddress string) (uuid.UUID, error)
	UpdateOnlineState(userID int, state bool) error
	RecoverPassword(recoveryUUID uuid.UUID, password string) (string, error)

	// Queries
	CompareCredentials(userID int, password string) (int, error)
	GetUserForID(userID int) (User, error)
	GetUserForName(name string) (User, error)
	GetUsersForPrefix(prefix string, limit int) ([]User, error)
	SelectRecoveryTokenIssueDate(recoveryUUID uuid.UUID) (time.Time, error)

	// Internal
	DeleteUser(userid int) error
}

// MessageRepo
type MessageRepo interface {

	// Mutations
	StoreTextMessage(conversation, user int, m TextMessage) (int, error)
	StoreCodeMessage(conversation, user int, m CodeMessage) (int, error)
	SetReadFlags(userid, conversationID int) error
	UpdateCode(messageID int, newCode, title, language string) error
	SetLockedSateForCodeMessage(messageID int, lockingUserID int) error

	// Queries
	FindForConversation(conversationID, beforeInSequence, limit int) ([]interface{}, error)
	FindCodeMessagesForConversation(conversationID, beforeInSequence, limit int) ([]interface{}, error)
	FindTextMessagesForConversation(conversationID, beforeInSequence, limit int) ([]interface{}, error)
	FindCodeMessageForID(messageID, conversationID int) (CodeMessage, error)
	FindTextMessageForID(messageID, conversationID int) (TextMessage, error)
	FindMessageStubForConversation(conversationID, messageID int) (Message, error)
	FindAllProgrammingLanguages() ([]ProgrammingLanguage, error)
}
