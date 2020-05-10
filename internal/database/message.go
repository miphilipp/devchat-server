package database

import (
	"sort"

	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

type messageRepository struct {
	db *pg.DB
}

func (r *messageRepository) SetReadFlags(userid int, conversationID int) error {
	_, err := r.db.Exec(
		`UPDATE message_status 
		 SET hasRead = true 
		 WHERE userid = ? AND conversationid = ?;`, userid, conversationID)
	return core.NewDataBaseError(err)
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
	ID   int
}

func (r *messageRepository) FindForConversation(
	conversationID int,
	beforeInSequence int,
	limit int) ([]interface{}, error) {
	stubs := make([]messageStub, 0, 10)
	_, err := r.db.Query(&stubs,
		`SELECT type, id
		FROM message
		WHERE conversationid = ? AND id < ? AND iscomplete = true
		ORDER BY id desc
		LIMIT ?;`, conversationID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	var largestID = getLargestID(stubs)
	codeMessages := make([]core.CodeMessage, 0, 10)
	_, err = r.db.Query(&codeMessages,
		`SELECT type, id, sentdate, author, code, language, title, lockedby
		FROM v_code_message
		WHERE conversationid = ? AND id <= ? AND id < ?
		ORDER BY id desc
		LIMIT ?;`, conversationID, largestID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	textMessages := make([]core.TextMessage, 0, 10)
	_, err = r.db.Query(&textMessages,
		`SELECT type, id, sentdate, author, text
		FROM v_text_message
		WHERE conversationid = ? AND id <= ? AND id < ?
		ORDER BY id desc
		LIMIT ?;`, conversationID, largestID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	mediaMessages := make([]core.MediaMessage, 0, 10)
	_, err = r.db.Query(&mediaMessages,
		`SELECT m.type, m.id, m.sentdate, m.author, m.text
		FROM v_media_message m
		JOIN media_object mo ON mo.message = m.id
		WHERE m.conversationid = ? AND m.id <= ? AND m.id < ? AND m.iscomplete = true
		GROUP BY m.id, m.sentdate, m.author, m.text, m.type
		HAVING COUNT(mo.message) > 0
		ORDER BY id desc
		LIMIT ?;`, conversationID, largestID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	for i := range mediaMessages {
		mediaObjects := make([]core.MediaObject, 0)
		_, err = r.db.Query(&mediaObjects,
			`SELECT name, id, filetype, meta FROM media_object WHERE message = ?;`, mediaMessages[i].ID)
		if err != nil {
			return make([]interface{}, 0), core.NewDataBaseError(err)
		}
		mediaMessages[i].Files = make([]core.MediaObject, len(mediaObjects))
		copy(mediaMessages[i].Files, mediaObjects)
	}

	messages := make([]interface{ core.Sequencable }, 0, len(codeMessages)+len(textMessages)+len(mediaMessages))
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
	for _, m := range mediaMessages {
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

func (r *messageRepository) FindMessageStubForConversation(conversationID int, messageID int) (core.Message, error) {
	var message core.Message
	_, err := r.db.QueryOne(&message,
		`SELECT m.type, m.id, m.sentdate, u.name as Author
		FROM message m
		JOIN public.user u ON m.userid = u.id
		WHERE m.conversationid = ? AND m.id = ?;`, conversationID, messageID)
	if err == pg.ErrNoRows {
		return message, core.ErrRessourceDoesNotExist
	}

	if err != nil {
		return message, core.NewDataBaseError(err)
	}

	return message, nil
}

// NewMessageRepository creates new instance of a type that implements core.MessageRepo
func NewMessageRepository(dbSession *pg.DB) core.MessageRepo {
	return &messageRepository{db: dbSession}
}

func (r *messageRepository) SetLockedSateForCodeMessage(messageID, lockingUserID int) error {
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

func (r *messageRepository) DeleteMessage(id int) error {
	_, err := r.db.Exec(
		`DELETE FROM public.message WHERE id = ?;`, id)
	return core.NewDataBaseError(err)
}

func (r *messageRepository) UpdateCompleteFlag(id int) error {
	_, err := r.db.Exec(
		`UPDATE public.message SET iscomplete = true WHERE id = ?;`, id)
	return core.NewDataBaseError(err)
}
