package database

import (
	//"fmt"
	"errors"
	"github.com/go-pg/pg/v9"
	core "github.com/miphilipp/devchat-server/internal"
)

type conversationRepository struct {
	db *pg.DB
}

func (r *conversationRepository) FindConversationsForUser(user int) ([]core.Conversation, error) {
	conversations := make([]core.Conversation, 0, 2)
	_, err := r.db.Query(&conversations, `
			SELECT id, title, repourl, calculateunreadmessages(id, ?) as unreadMessagesCount
			FROM conversation c
			JOIN group_association g on c.id = g.conversationid
			WHERE userid = ?;`, user, user)

	return conversations, core.NewDataBaseError(err)
}

func (r *conversationRepository) SetAdminState(userID, conversationID int, state bool) error {
	_, err := r.db.Exec(
		`UPDATE group_association
		SET isAdmin = ?
		WHERE 
			userid = ? AND 
			conversationid = ? AND 
			joined IS NOT NULL AND
			hasleft = false;`, state, userID, conversationID)

	return core.NewDataBaseError(err)
}

func (r *conversationRepository) FindConversations() ([]core.Conversation, error) {
	conversations := make([]core.Conversation, 0, 2)
	_, err := r.db.Query(&conversations, `
			SELECT id as ID, title as Title, repourl as Repourl FROM conversation;`)
	
	return conversations, core.NewDataBaseError(err)
}

// CreateConversation inserts a conversation into the database
func (r *conversationRepository) CreateConversation(userCtx int, c core.Conversation, initialMembers []int) (core.Conversation, error) {
	var insertedID = -1
	emptyConverstion := core.Conversation{}

	_, err := callFunction(r.db, "createConversation", &insertedID, userCtx, c.Title, c.Repourl, pg.Array(initialMembers))
	if err != nil {
		return emptyConverstion, err
	}

	return core.Conversation{
		Title: c.Title,
		ID: insertedID,         
		Repourl: c.Repourl,    
	}, nil
}

// DeleteConveration deletes a conversation from the database
func (r *conversationRepository) DeleteConversation(id int) error {
	res, err := r.db.Exec("DELETE FROM conversation WHERE id = ?;", id)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrNothingChanged
	}

	return nil
}

func (r *conversationRepository) FindInvitations(userCtx int) ([]core.Invitation, error) {
	model := []struct {
		ConversationID    int	 `pg:"conversationid"`
		ConversationTitle string `pg:"conversationtitle"`
		Recipient		  int	 `pg:"recipient"`
	}{}

	_, err := r.db.Query(&model,
		`SELECT conversationId, conversationTitle, recipient FROM v_invitation WHERE recipient = ?;`, 
	userCtx)
	if err != nil {
		return nil, core.NewDataBaseError(err)
	}

	invitations := make([]core.Invitation, len(model))
	for i := 0; i < len(model); i++ {
		invitations[i].ConversationID = model[i].ConversationID
		invitations[i].ConversationTitle = model[i].ConversationTitle
		invitations[i].Recipient = model[i].Recipient
	}

	return invitations, nil
}

func (r *conversationRepository) MarkAsInvited(userID int, conversationID int) error {
	_, err := callStoredProcedure(r.db, "inviteUser", userID, conversationID)
	return core.NewDataBaseError(err)
}

func (r *conversationRepository) RemoveGroupAssociation(userID int, conversationID int) error {
	_, err := r.db.Exec(
		`DELETE FROM group_association 
		WHERE userid = ? AND conversationid = ?;`, userID, conversationID)
	return core.NewDataBaseError(err)
}

func (r *conversationRepository) SetAsLeft(userID int, conversationID int) error {
	_, err := r.db.Exec(
		`UPDATE group_association 
		SET hasleft = true, joined = null, isadmin = false
		WHERE userid = ? AND conversationid = ?;`, userID, conversationID)
	return core.NewDataBaseError(err)
}

func (r *conversationRepository) MarkAsJoined(userID int, conversationID int) (int, error) {
	var newColorIndex int
	_, err := callFunction(r.db, "joinConversation", &newColorIndex, userID, conversationID)
	return newColorIndex, core.NewDataBaseError(err)
}

func (r *conversationRepository) FindConversationForID(conversationID int) (core.Conversation, error) {
	c := struct {
		ID int
		Title string
		RepoURL string `pg:"repourl"`
	}{}
	_, err := r.db.QueryOne(&c, 
		`SELECT id, title, repourl FROM public.conversation WHERE id = ?;`, 
	conversationID)
	if errors.Is(err, pg.ErrNoRows) {
		return core.Conversation{}, core.ErrConversationDoesNotExist
	}

	if err != nil {
		return core.Conversation{}, core.NewDataBaseError(err)
	}

	return core.Conversation{
		Title: c.Title,
		ID: c.ID,
		Repourl: c.RepoURL,
	}, nil
}

func (r *conversationRepository) SetMetaDataOfConversation(conversation core.Conversation) error {
	_, err := r.db.ExecOne( 
		`UPDATE public.conversation
		SET title = ?, repourl = ?
		WHERE id = ?;`, 
		conversation.Title, conversation.Repourl, conversation.ID)
	if err == pg.ErrNoRows {
		return core.ErrConversationDoesNotExist
	}

	return core.NewDataBaseError(err)
}

func (r *conversationRepository) GetUsersInConversation(conversationID int) ([]core.UserInConversation, error) {
	users := make([]core.UserInConversation, 0, 5)
	_, err := r.db.Query(&users,
		`SELECT name, id, isadmin, colorIndex, hasjoined, hasleft, isdeleted
		FROM v_every_member
		WHERE conversationId = ?;`,
		conversationID)
	if err != nil {
		return nil, core.NewDataBaseError(err)
	}
	return users, nil
}

// IsUserInConversation returns true if the user is a member of the specified conversation.
func (r *conversationRepository) IsUserInConversation(userID int, conversationID int) (bool, error) {
	var res int
	_, err := r.db.Query(&res,
		`SELECT COUNT(*) FROM v_joined_member WHERE userid = ? AND conversationId = ?;`, 
	userID, conversationID)
	if err != nil {
		return false, core.NewDataBaseError(err)
	}

	return res == 1, nil
}

func (r *conversationRepository) IsUserAdminOfConveration(userID int, conversationID int) (bool, error) {
	var res int
	_, err := r.db.Query(&res,
		`SELECT COUNT(*) FROM v_admin WHERE userid = ? AND conversationId = ?;`, 
	userID, conversationID)
	if err != nil {
		return false, core.NewDataBaseError(err)
	}

	return res == 1, nil
}

func (r *conversationRepository) CountAdminsOfConversation(conversationID int) (int, error)  {
	var numberOfAdmins int
	_, err := r.db.QueryOne(&numberOfAdmins,
		`SELECT count(*) FROM v_admin WHERE conversationId = ?;`,
		conversationID)
	if err != nil {
		return 0, core.NewDataBaseError(err)
	}
	return numberOfAdmins, nil
}

func NewConversationRepository(dbSession *pg.DB) core.ConversationRepo {
	return &conversationRepository{db: dbSession}
}
