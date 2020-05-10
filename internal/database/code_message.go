package database

import (
	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

func (r *messageRepository) StoreCodeMessage(conversation int, user int, m core.CodeMessage) (int, error) {
	var id = -1
	_, err := callFunction(r.db, "createCodeMessage", &id, user, conversation, m.Code, m.Sentdate, m.Language, m.Title)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}
	return id, nil
}

func (r *messageRepository) FindCodeMessagesForConversation(
	conversationID int,
	beforeInSequence int,
	limit int) ([]interface{}, error) {

	codeMessages := make([]core.CodeMessage, 0, 10)
	_, err := r.db.Query(&codeMessages,
		`SELECT type, id, sentdate, author, code, language, title, lockedby
		FROM v_code_message
		WHERE conversationid = ? AND id < ?
		ORDER BY id desc
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
		`SELECT type, id, sentdate, author, code, language, title, lockedby
		FROM v_code_message
		WHERE id = ? AND conversationid = ?;`, messageID, conversationID)
	if err != nil && err == pg.ErrNoRows {
		return core.CodeMessage{}, core.ErrRessourceDoesNotExist
	}

	if err != nil {
		return core.CodeMessage{}, core.NewDataBaseError(err)
	}

	return message, nil
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
