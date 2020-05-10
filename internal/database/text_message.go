package database

import (
	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

// CreateMessage adds a message to the database
func (r *messageRepository) StoreTextMessage(conversation int, user int, m core.TextMessage) (int, error) {
	var id = -1
	_, err := callFunction(r.db, "createTextMessage", &id, user, conversation, m.Sentdate, m.Text)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}

	return id, nil
}

func (r *messageRepository) FindTextMessageForID(messageID, conversationID int) (core.TextMessage, error) {
	var message core.TextMessage
	_, err := r.db.QueryOne(&message,
		`SELECT type, id, sentdate, author, text
		FROM v_text_message
		WHERE id = ? AND conversationid = ?;`, messageID, conversationID)
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
		`SELECT type, id, sentdate, author, text
		FROM v_text_message m
		WHERE conversationid = ? AND id < ?
		ORDER BY id desc
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
