package core

import (
	"time"
	"github.com/google/uuid"
)

// MessageType describes an MessageType enum value.
type MessageType int

const UintSize = 32 << (^uint(0) >> 32 & 1) // 32 or 64

const (
    MaxInt  = 1<<(UintSize-1) - 1
    MinInt  = -MaxInt - 1
    MaxUint = 1<<UintSize - 1
)

const (
	// TextMessageType represents the type of a pure text message.
	TextMessageType MessageType = 0

	// CodeMessageType represents the type of a code message.
	CodeMessageType MessageType = 1

	// UndefinedMesssageType represents a message of any kind.
	UndefinedMesssageType MessageType = -1
)

// Cronological is used to enable chronological sorting
type Cronological interface {
	GetDate() time.Time
}

type Sequencable interface {
	GetSequenceNumber() int
}

// Conversation
type Conversation struct {
	Title       	string `json:"title"`
	ID          	int	   `json:"id"`
	Repourl     	string `json:"repoUrl"`
	NUnreadMessages int	   `json:"nUnreadMessages" pg:"unreadmessagescount"`
}

// MailingService provides an simple interface to send emails.
type MailingService interface {
	SendEmail(to, subject, body string) error
}

// Invitation
type Invitation struct {
	ConversationID    int	 `json:"conversationId"`
	ConversationTitle string `json:"conversationTitle"`
	Recipient		  int	 `json:"recipient"`
}

// Message is the abstract base type of any message.
type Message struct {
	ID        		int			`json:"id"`
	Type      		MessageType	`json:"type"`
	Sentdate  		time.Time   `json:"sentdate"`
	ProvisionaryID  int			`json:"provisionaryId,omitempty"`
	Author	  		string 		`json:"author"`
}

// TextMessage is derived from Message.
type TextMessage struct {
	Message
	Text string `json:"text"`
}

// CodeMessage is derived from Message.
type CodeMessage struct {
	Message
	Code     string	`json:"code"`
	Language string `json:"language"`
	Title	 string `json:"title"`
	LockedBy int	`json:"lockedBy" pg:"lockedby"`
}

// GetDate makes Message implement the Cronological interface.
func (m Message) GetDate() time.Time  {
	return m.Sentdate
}

// GetDate makes CodeMessage implement the Cronological interface.
func (m CodeMessage) GetDate() time.Time  {
	return m.Sentdate
}

// GetDate makes TextMessage implement the Cronological interface.
func (m TextMessage) GetDate() time.Time  {
	return m.Sentdate
}



// GetSequenceNumber makes Message implement the Sequencable interface.
func (m Message) GetSequenceNumber() int  {
	return m.ID
}

// GetSequenceNumber makes CodeMessage implement the Sequencable interface.
func (m CodeMessage) GetSequenceNumber() int  {
	return m.ID
}

// GetSequenceNumber makes TextMessage implement the Sequencable interface.
func (m TextMessage) GetSequenceNumber() int  {
	return m.ID
}

// User
type User struct {
	ID                  int	      `pg:"id" json:"id"`
	Email               string    `pg:"email" json:"email,omitempty"`
	Name                string	  `pg:"name" json:"name"`
	ConfirmationUUID	uuid.UUID `json:"-"`
	LockedOutSince      time.Time `json:"-"`
	FailedLoginAttempts int       `json:"-"`
	LastFailedLogin		time.Time `json:"-"`
	IsDeleted 			bool  	  `pg:"isdeleted" json:"isDeleted"`
}

// UserInConversation
type UserInConversation struct {
	User
	IsAdmin    bool	`pg:"isadmin" json:"isAdmin"`
	ColorIndex int	`pg:"colorindex" json:"colorIndex"`
	HasJoined  bool `pg:"hasjoined" json:"hasJoined"`
	HasLeft	   bool `pg:"hasleft" json:"hasLeft"`
}

// ProgrammingLanguage
type ProgrammingLanguage struct {
	Name 		string 	`json:"name"`
	IsRunnable 	bool 	`pg:"IsRunnable" json:"isRunnable`
}