package database

import (
	//"fmt"
	"sort"

	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

type messageRepository struct {
	db *pg.DB
}

// CreateMessage adds a message to the database
func (r *messageRepository) StoreTextMessage(conversation int, user int, m core.TextMessage) (int, error) {
	var id = -1
	_, err := callFunction(r.db, "createTextMessage", &id, user, conversation, m.Sentdate, m.Text)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}

	return id, nil
}

func (r *messageRepository) StoreCodeMessage(conversation int, user int, m core.CodeMessage) (int, error) {
	var id = -1
	_, err := callFunction(r.db, "createCodeMessage", &id, user, conversation,  m.Code, m.Sentdate, m.Language, m.Title)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}
	return id, nil
}

func (r *messageRepository) UpdateCode(messageID int, newCode, title, language string) error {
	res, err := r.db.Exec(
		`UPDATE public.code_message
		 SET code = ?, language = ?, title = ?
		 WHERE id = ?;`, newCode, language, title, messageID)
	if err != nil {
		return core.NewDataBaseError(err)
	}
	
	if res.RowsAffected() == 0 {
		return core.ErrRessourceDoesNotExist
	}
	return nil
}

func (r *messageRepository) SetReadFlags(userid int, conversationID int) error {
	res, err := r.db.Exec(
		`UPDATE message_status 
		 SET hasRead = true 
		 WHERE userid = ? AND conversationid = ?;`, userid, conversationID)
	if err != nil {
		return core.NewDataBaseError(err)
	}
	
	if res.RowsAffected() == 0 {
		return core.ErrRessourceDoesNotExist
	}
	return nil	
}


func containsID(stubs []messageStub, id int) bool {
    for _, s := range stubs {
        if s.ID == id {
            return true
        }
    }
    return false
}

func getLargestID(arr []messageStub) int {
	if len(arr) == 0 {
		return 0
	}

	largestID := arr[0].ID
    for _, s := range arr {
        if s.ID > largestID {
			largestID = s.ID
        }
    }
    return largestID
}

type messageStub struct {
	Type core.MessageType
	ID int
}

func (r *messageRepository) FindForConversation(
	conversationID int, 
	beforeInSequence int, 
	limit int) ([]interface{}, error) {
	stubs := make([]messageStub, 0, 10)
	_, err := r.db.Query(&stubs,
		`SELECT m.type, m.id
		FROM message m
		JOIN conversation co ON m.conversationId = co.id
		WHERE m.conversationid = ? AND m.id < ?
		ORDER BY m.id desc
		LIMIT ?;`, conversationID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	var largestID = getLargestID(stubs)
	codeMessages := make([]core.CodeMessage, 0, 10)
	_, err = r.db.Query(&codeMessages,
		`SELECT m.type, m.id, m.sentdate, m.author, m.code, m.language, m.title, m.lockedby
		FROM v_code_message m
		JOIN conversation co ON m.conversationId = co.id
		WHERE m.conversationid = ? AND m.id <= ? AND m.id < ?
		ORDER BY m.id desc
		LIMIT ?;`, conversationID, largestID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	textMessages := make([]core.TextMessage, 0, 10)
	_, err = r.db.Query(&textMessages,
		`SELECT m.type, m.id, m.sentdate, m.author, m.text
		FROM v_text_message m
		JOIN conversation co ON m.conversationId = co.id
		WHERE m.conversationid = ? AND m.id <= ? AND m.id < ?
		ORDER BY m.id desc
		LIMIT ?;`, conversationID, largestID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	messages := make([]interface{core.Sequencable}, 0, len(codeMessages) + len(textMessages))
	for _, m := range codeMessages {
		if containsID(stubs, m.ID) {
			messages = append(messages, m)
		}
	}
	for _, m := range textMessages {
		if containsID(stubs, m.ID) {
			messages = append(messages, m)
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].GetSequenceNumber() < messages[j].GetSequenceNumber()
	})

	messagesI := make([]interface{}, len(messages))
	for i := range messages {
		messagesI[i] = messages[i]
	}

	return messagesI, nil
}

func (r *messageRepository) FindCodeMessagesForConversation(
	conversationID int, 
	beforeInSequence int, 
	limit int) ([]interface{}, error) {

	codeMessages := make([]core.CodeMessage, 0, 10)
	_, err := r.db.Query(&codeMessages,
		`SELECT m.type, m.id, m.sentdate, m.author, m.code, m.language, m.title, m.lockedby
		FROM v_code_message m
		JOIN conversation co ON m.conversationId = co.id
		WHERE m.conversationid = ? AND m.id < ?
		ORDER BY m.id desc
		LIMIT ?;`, conversationID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	messagesI := make([]interface{}, len(codeMessages))
	for i := range codeMessages {
		messagesI[i] = codeMessages[i]
	}

	return messagesI, nil
}

func (r *messageRepository) FindCodeMessageForID(messageID, conversationID int) (core.CodeMessage, error) {
	var message core.CodeMessage
	_, err := r.db.QueryOne(&message,
		`SELECT m.type, m.id, m.sentdate, m.author, m.code, m.language, m.title, m.lockedby
		FROM v_code_message m
		JOIN conversation c ON c.id = m.conversationid
		WHERE m.id = ? AND c.id = ?;`, messageID, conversationID)
	if err != nil && err == pg.ErrNoRows {
		return core.CodeMessage{}, core.ErrRessourceDoesNotExist
	}

	if err != nil {
		return core.CodeMessage{}, core.NewDataBaseError(err)
	}

	return message, nil
}

func (r *messageRepository) FindTextMessageForID(messageID, conversationID int) (core.TextMessage, error) {
	var message core.TextMessage
	_, err := r.db.QueryOne(&message,
		`SELECT m.type, m.id, m.sentdate, m.author, m.text
		FROM v_text_message m
		JOIN conversation c ON c.id = m.conversationid
		WHERE m.id = ? AND c.id = ?;`, messageID, conversationID)
	if err != nil && err == pg.ErrNoRows {
		return core.TextMessage{}, core.ErrRessourceDoesNotExist
	}

	if err != nil {
		return core.TextMessage{}, core.NewDataBaseError(err)
	}

	return message, nil
}

func (r *messageRepository) FindTextMessagesForConversation(
	conversationID int, 
	beforeInSequence int, 
	limit int) ([]interface{}, error) {

	textMessages := make([]core.TextMessage, 0, 10)
	_, err := r.db.Query(&textMessages,
		`SELECT m.type, m.id, m.sentdate, m.author, m.text
		FROM v_text_message m
		JOIN conversation co ON m.conversationId = co.id
		WHERE c.conversationid = ? AND m.id < ?
		ORDER BY m.id desc
		LIMIT ?;`, conversationID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), err
	}

	messagesI := make([]interface{}, len(textMessages))
	for i := range textMessages {
		messagesI[i] = textMessages[i]
	}

	return messagesI, nil
}

func (r *messageRepository) FindMessageStubForConversation(conversationID int, messageID int) (core.Message, error)  {
	var message core.Message
	_, err := r.db.QueryOne(&message,
		`SELECT m.type, m.id, m.sentdate, u.name as "Author"
		FROM message m
		JOIN conversation co ON m.conversationId = co.id
		JOIN public.user u ON m.userid = u.id
		WHERE co.id = ? AND m.id = ?;`, conversationID, messageID)
	if err == pg.ErrNoRows {
		return message, core.ErrRessourceDoesNotExist
	}

	if err != nil {
		return message, core.NewDataBaseError(err)
	}

	return message, nil
}

func (r *messageRepository) FindAllProgrammingLanguages() ([]core.ProgrammingLanguage, error) {
	languages := make([]core.ProgrammingLanguage, 0, 35)
	_, err := r.db.Query(&languages,
		`SELECT name, runnable as "IsRunnable" 
		 FROM programming_language 
		 ORDER BY name;`)
	if err != nil {
		return nil, core.NewDataBaseError(err)
	}
	return languages, nil
}

// NewMessageRepository creates new instance of a type that implements core.MessageRepo
func NewMessageRepository(dbSession *pg.DB) core.MessageRepo {
	return &messageRepository{db: dbSession}
}

func (r *messageRepository) SetLockedSateForCodeMessage(messageID, lockingUserID int) error  {
	var id interface{}
	if lockingUserID > 0 {
		id = lockingUserID
	} else {
		id = nil
	}
	res, err := r.db.Exec(
		`UPDATE public.code_message
		 SET lockedby = ? 
		 WHERE id = ?;`, id, messageID)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrRessourceDoesNotExist
	}

	return nil
}
