package database

import (
	//"fmt"
	"strings"
	"time"
	"errors"
	"github.com/go-pg/pg/v9"
	"github.com/google/uuid"
	core "github.com/miphilipp/devchat-server/internal"
)

type userRepository struct {
	db *pg.DB
}

// AuthenticateUser authenticates a user and returns its id if the authentication
// was successfull
func (r *userRepository) CompareCredentials(userID int, password string) (int, error) {
	var id int
	_, err := r.db.QueryOne(&id,
		"SELECT id FROM public.user WHERE id = ? AND password = crypt(?, password);",
		userID, password)

	if errors.Is(err, pg.ErrNoRows)  {
		return -1, nil
	}

	if err != nil {
		return -1, core.NewDataBaseError(err)
	}

	return id, nil
}

func (r *userRepository) IncrementFailedLoginAttempts(user string) error {
	res, err := r.db.Exec(
		`UPDATE public.user 
		 SET failedLoginAttempts = failedLoginAttempts + 1,
		 lastfailedlogin = current_timestamp at time zone 'utc'
		 WHERE name = ? AND isdeleted = false;`, user)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrUserDoesNotExist
	}
	
	return nil
}

func (r *userRepository) DeleteUser(userid int) error {
	_, err := r.db.Exec(`
		DELETE FROM public.user 
		WHERE id = ?;`, userid)

	return core.NewDataBaseError(err)
}


// DeleteUser sets the deleted flag to true and clears all sensetive user data.
func (r *userRepository) SoftDeleteUser(userID int) error {
	_, err := callStoredProcedure(r.db, "deleteAccount", userID)
	return core.NewDataBaseError(err)
}

// CreateUser adds new user to the database
func (r *userRepository) CreateUser(user core.User, password string) (core.User, error) {
	userOutput := struct {
		Name  			 string
		Email 			 string
		ID    			 int
		ConfirmationUUID uuid.UUID `pg: "confirmation_uuid"`
		//languageCode string TODO: Implementieren
	}{}
	_, err := r.db.QueryOne(&userOutput,
		`INSERT INTO public.user (name, email, password, confirmation_uuid) 
		 VALUES (?, ?, crypt(?, gen_salt('bf')), uuid_generate_v4())
		 RETURNING id, name, email, confirmation_uuid;`, 
		 user.Name, user.Email, password)

	if err != nil {
		return core.User{}, core.NewDataBaseError(err)
	}
	return core.User{
		ID:    userOutput.ID,
		Name:  userOutput.Name,
		Email: userOutput.Email,
		ConfirmationUUID: userOutput.ConfirmationUUID,
		//LanguageCode: userOutput.languageCode,
	}, err
}

func (r *userRepository) RecoverPassword(recoveryUUID uuid.UUID, password string) (string, error) {
	var username string 
	_, err := r.db.QueryOne(&username,
		`UPDATE public.user
		SET 
			password = crypt(?, gen_salt('bf')), 
			recovery_uuid = NULL, 
			recovery_uuid_issue_date = NULL
		WHERE recovery_uuid = ? AND isdeleted = false
		RETURNING name;`, password, recoveryUUID)
	if err != nil && err == pg.ErrNoRows {
		return username, core.ErrInvalidToken
	}

	if err != nil {
		return username, core.NewDataBaseError(err)
	}
	return username, nil
}

func (r *userRepository) SelectRecoveryTokenIssueDate(recoveryUUID uuid.UUID) (time.Time, error)  {
	var recoveryTokenIssueDate time.Time 
	_, err := r.db.QueryOne(&recoveryTokenIssueDate,
		`SELECT recovery_uuid_issue_date
		FROM public.user 
		WHERE recovery_uuid = ? AND isdeleted = false;`, recoveryUUID)
	if err != nil && err == pg.ErrNoRows {
		return recoveryTokenIssueDate, core.ErrInvalidToken
	}

	if err != nil {
		return recoveryTokenIssueDate, core.NewDataBaseError(err)
	}
	return recoveryTokenIssueDate, nil
}

func (r *userRepository) CreateRecoverID(emailAddress string) (uuid.UUID, error) {
	var updatedUUID uuid.UUID
	_, err := r.db.QueryOne(&updatedUUID,
		`UPDATE public.user 
		 SET recovery_uuid = uuid_generate_v4(), recovery_uuid_issue_date = current_timestamp at time zone 'utc'
		 WHERE email = ?
		 RETURNING recovery_uuid;`, emailAddress)

	if errors.Is(err, pg.ErrNoRows)  {
		return updatedUUID, core.ErrUserDoesNotExist
	}

	if err != nil {
		return updatedUUID, core.NewDataBaseError(err)
	}
	
	return updatedUUID, nil
}

func (r *userRepository) SetPassword(user int, newPassword string) error {
	res, err := r.db.Exec(
		`UPDATE public.user SET password = crypt(?, gen_salt('bf'))
		 WHERE id = ?;`, newPassword, user)
	if err != nil {
		return core.NewDataBaseError(err)
	}
	
	if res.RowsAffected() == 0 {
		return core.ErrUserDoesNotExist
	}
	return nil
}

func (r *userRepository) GetUserForID(userID int) (core.User, error) {
	userOutput := struct {
		Name  string
		Email string
		ID    int
		//languageCode string TODO: Implementieren
	}{}
	_, err := r.db.QueryOne(&userOutput,
		"SELECT name, email, id FROM public.user WHERE id = ? AND isdeleted = false;", userID)
	if err != nil && err == pg.ErrNoRows {
		return core.User{}, core.ErrUserDoesNotExist
	}

	if err != nil {
		return core.User{}, core.NewDataBaseError(err)
	}
	return core.User{
		ID:    userOutput.ID,
		Name:  userOutput.Name,
		Email: userOutput.Email,
		//LanguageCode: userOutput.languageCode,
	}, nil
}

func (r *userRepository) SetConfirmationIDToNULL(token string) (string, error)  {
	var username string
	_, err := r.db.QueryOne(&username,
		`UPDATE public.user SET confirmation_uuid = NULL
		 WHERE confirmation_uuid = ?
		 RETURNING name;`, token)
	if err != nil {
		return "", core.NewDataBaseError(err)
	}
	
	return username, nil
}

func (r *userRepository) GetUserForName(name string) (core.User, error) {
	userOutput := struct {
		Name  				string
		Email 				string
		ID    				int
		FailedLoginAttempts int 		`pg:"failedloginattempts"`
		LockedOutSince 		time.Time 	`pg:"lockedoutsince"`
		LastFailedLogin 	time.Time	`pg:"lastfailedlogin"`
		IsDeleted			bool		`pg:"isdeleted"`
		//languageCode string TODO: Implementieren
	}{}
	_, err := r.db.QueryOne(&userOutput,
		`SELECT name, email, id, failedloginattempts, lockedoutsince, lastfailedlogin, isdeleted
		 FROM public.user WHERE name = ?;`, name)
	if err != nil && err == pg.ErrNoRows {
		return core.User{}, core.ErrUserDoesNotExist
	}

	if err != nil {
		return core.User{}, core.NewDataBaseError(err)
	}
	return core.User{
		ID:    userOutput.ID,
		Name:  userOutput.Name,
		Email: userOutput.Email,
		LockedOutSince: userOutput.LockedOutSince,
		FailedLoginAttempts: userOutput.FailedLoginAttempts,
		LastFailedLogin: userOutput.LastFailedLogin,
		IsDeleted: userOutput.IsDeleted,
		//LanguageCode: userOutput.languageCode,
	}, nil
}

func (r *userRepository) GetUsersForPrefix(prefix string, limit int) ([]core.User, error) {
	users := make([]core.User, 0, 10)
	_, err := r.db.Query(&users,
		`SELECT id, name 
		FROM public.user 
		WHERE lower(name) LIKE ?||'%' AND isdeleted = false LIMIT ?;`,
		strings.ToLower(prefix), limit)
	if err != nil {
		return nil, core.NewDataBaseError(err)
	}
	return users, nil
}

func (r *userRepository) LockUser(userID int) error {
	res, err := r.db.Exec(
		`UPDATE public.user 
		SET failedLoginAttempts = 0, lockedOutSince = current_timestamp at time zone 'utc' 
		WHERE id = ?;`,
		userID)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrUserDoesNotExist
	}

	return nil
}

func (r *userRepository) UnlockUser(userID int) error {
	res, err := r.db.Exec(
		`UPDATE public.user 
		SET lockedOutSince = NULL, failedLoginAttempts = 0 
		WHERE id = ?;`,
		userID)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrUserDoesNotExist
	}
	
	return nil
}

func (r *userRepository) UpdateOnlineState(userID int, state bool) error  {
	res, err := r.db.Exec(
		`UPDATE public.user 
		SET isonline = ? 
		WHERE id = ?;`,
		userID, state)
	if err != nil {
		return core.NewDataBaseError(err)
	}

	if res.RowsAffected() == 0 {
		return core.ErrUserDoesNotExist
	}
	
	return nil
}

// NewUserRepository creates new instance of a type that implements core.UserRepo
func NewUserRepository(dbSession *pg.DB) core.UserRepo {
	return &userRepository{db: dbSession}
}
