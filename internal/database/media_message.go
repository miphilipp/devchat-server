package database

import (
	"encoding/json"

	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

func (r *messageRepository) StoreMediaMessage(conversation, user int, m core.MediaMessage) (int, error) {
	var id = -1
	_, err := callFunction(r.db, "createMediaMessage", &id, user, conversation, m.Sentdate, m.Text)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}

	return id, nil
}

func (r *messageRepository) CreateMediaObject(messageID int, name, fileType string) (int, error) {
	var mediaID int
	_, err := r.db.QueryOne(&mediaID,
		`INSERT INTO media_object(message, name, filetype)
		VALUES(?, ?, ?)
		RETURNING id;`, messageID, name, fileType)

	if err != nil {
		return 0, core.NewDataBaseError(err)
	}

	return mediaID, nil
}

func (r *messageRepository) FindMediaMessagesForConversation(
	conversationID int,
	beforeInSequence int,
	limit int) ([]interface{}, error) {

	mediaMessages := make([]core.MediaMessage, 0, 10)
	_, err := r.db.Query(&mediaMessages,
		`SELECT type, m.id, sentdate, author, Text
		FROM v_media_message m
		WHERE conversationid = ? AND id < ? AND m.iscomplete = true
		ORDER BY id desc
		LIMIT ?;`, conversationID, beforeInSequence, limit)
	if err != nil {
		return make([]interface{}, 0), core.NewDataBaseError(err)
	}

	for _, message := range mediaMessages {
		mediaObjects := make([]core.MediaObject, 0)
		_, err = r.db.Query(&mediaObjects,
			`SELECT name, id, filetype, meta FROM media_object WHERE message = ?;`, message.ID)
		if err != nil {
			return make([]interface{}, 0), core.NewDataBaseError(err)
		}
		message.Files = mediaObjects
	}

	messagesI := make([]interface{}, len(mediaMessages))
	for i := range mediaMessages {
		messagesI[i] = mediaMessages[i]
	}

	return messagesI, nil
}

func (r *messageRepository) FindMediaMessageForID(messageID, conversationID int) (core.MediaMessage, error) {
	var message core.MediaMessage
	_, err := r.db.QueryOne(&message,
		`SELECT type, id, sentdate, author, text
		FROM v_media_message
		WHERE id = ? AND conversationid = ?;`, messageID, conversationID)
	if err != nil && err == pg.ErrNoRows {
		return core.MediaMessage{}, core.ErrRessourceDoesNotExist
	}

	mediaObjects := make([]core.MediaObject, 0)
	_, err = r.db.Query(&mediaObjects, `SELECT name, id, filetype, meta FROM media_object WHERE message = ?;`, message.ID)
	if err != nil {
		return core.MediaMessage{}, core.NewDataBaseError(err)
	}
	message.Files = mediaObjects

	if err != nil {
		return core.MediaMessage{}, core.NewDataBaseError(err)
	}

	return message, nil
}

func (r *messageRepository) SetMetaOfMediaMessage(id int, meta interface{}) error {
	res, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE media_object SET meta = ? WHERE id = ?;`, string(res), id)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	return nil
}

func (r *messageRepository) FindMediaObjectForID(id, conversationID int) (core.MediaObject, error) {
	var obj core.MediaObject
	_, err := r.db.QueryOne(&obj,
		`SELECT mo.filetype, mo.name, mo.id, mo.meta
		FROM media_object mo
		JOIN v_media_message m ON m.id = mo.message
		WHERE mo.id = ? AND m.conversationid = ?;`, id, conversationID)
	if err != nil && err == pg.ErrNoRows {
		return core.MediaObject{}, core.ErrRessourceDoesNotExist
	}

	return obj, core.NewDataBaseError(err)
}
