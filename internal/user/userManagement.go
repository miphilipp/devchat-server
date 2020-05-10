package user

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	core "github.com/miphilipp/devchat-server/internal"
)

type Service interface {
	GetUserForName(name string) (core.User, error)
	SearchUsers(prefix string) ([]core.User, error)
	GetUserForID(id int) (core.User, error)

	AuthenticateUser(username, password string) (int, error)
	ChangePassword(userid int, oldPassword, newPassword string) error
	UpdateOnlineTimestamp(userCtx int) error

	DeleteAccount(userID int) error
	CreateAccount(newUser core.User, password, serverAddr string) error
	ConfirmAccount(token string) (string, error)

	ResetPassword(recoveryUUID, newPassword string) (string, error)
	SendPasswordResetMail(emailAddress, baseURL, language string) error

	SaveAvatar(userID int, pathPrefix, fileType string, buffer []byte) error
	DeleteAvatar(pathPrefix string, userID int) error
	GetAvatar(userID int, pathPrefix string, nodefault bool) (string, time.Time, error)
}

type Config struct {
	AllowSignup              bool
	LockOutTimeMinutes       time.Duration
	NLoginAttempts           int
	PasswordResetTimeMinutes time.Duration
}

type service struct {
	repo    core.UserRepo
	mailing core.MailingService
	cfg     Config
}

// NewService creates a new user managment service.
func NewService(repo core.UserRepo, mailing core.MailingService, cfg Config) Service {
	if cfg.LockOutTimeMinutes == 0 {
		cfg.LockOutTimeMinutes = 5
	}

	if cfg.NLoginAttempts == 0 {
		cfg.NLoginAttempts = 5
	}

	if cfg.PasswordResetTimeMinutes == 0 {
		cfg.PasswordResetTimeMinutes = 30
	}

	return &service{
		repo:    repo,
		mailing: mailing,
		cfg:     cfg,
	}
}

func (s *service) SaveAvatar(userID int, pathPrefix, fileType string, buffer []byte) error {
	if !strings.HasPrefix(fileType, "image/") {
		return core.ErrInvalidFileType
	}

	err := ioutil.WriteFile(path.Join(pathPrefix, strconv.Itoa(userID)), buffer, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) DeleteAvatar(pathPrefix string, userID int) error {
	err := os.Remove(path.Join(pathPrefix, strconv.Itoa(userID)))
	if err != nil {
		e, ok := err.(*os.PathError)
		if !ok || e.Err != syscall.ENOENT {
			return err
		}
	}
	return nil
}

func (s *service) UpdateOnlineTimestamp(userCtx int) error {
	return s.repo.UpdateOnlineState(userCtx)
}

func (s *service) AuthenticateUser(username, password string) (int, error) {
	user, err := s.repo.GetUserForName(username)
	if err != nil {
		return -1, err
	}

	if user.IsDeleted {
		return -1, core.ErrUserDoesNotExist
	}

	if user.ConfirmationUUID != uuid.Nil {
		return -1, core.ErrAccountNotConfirmed
	}

	if !user.LockedOutSince.IsZero() && user.LockedOutSince.Add(time.Minute*s.cfg.LockOutTimeMinutes).After(time.Now().UTC()) {
		return -1, core.ErrLockedOut
	}

	id, err := s.repo.CompareCredentials(user.ID, password)
	if err != nil {
		return -1, err
	}

	if id == -1 {
		if user.LastFailedLogin.IsZero() || user.LastFailedLogin.Add(time.Hour).After(time.Now().UTC()) {
			if user.FailedLoginAttempts+1 > s.cfg.NLoginAttempts {
				_ = s.repo.LockUser(user.ID)
				return -1, core.ErrLockedOut
			}
		} else {
			err = s.repo.UnlockUser(user.ID)
			if err != nil {
				return -1, err
			}
		}

		err = s.repo.IncrementFailedLoginAttempts(username)
		if err != nil {
			return -1, err
		}

		return -1, nil
	}

	err = s.repo.UnlockUser(user.ID)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (s *service) SearchUsers(prefix string) ([]core.User, error) {
	return s.repo.GetUsersForPrefix(prefix, 15)
}

func (s *service) GetUserForID(id int) (core.User, error) {
	return s.repo.GetUserForID(id)
}

func (s *service) GetUserForName(name string) (core.User, error) {
	return s.repo.GetUserForName(name)
}

func (s *service) DeleteAccount(userID int) error {
	return s.repo.SoftDeleteUser(userID)
}

func (s *service) CreateAccount(newUser core.User, password, serverAddr string) error {
	if s.cfg.AllowSignup == false {
		return core.ErrFeatureDeactivated
	}

	_, err := s.GetUserForName(newUser.Name)
	if err != nil && err != core.ErrUserDoesNotExist {
		return err
	}

	if len(newUser.Name) == 0 {
		return core.ErrAccessDenied
	}

	if err != core.ErrUserDoesNotExist {
		return core.ErrAlreadyExists
	}

	if checkPasswordPolicy(password) {
		return core.ErrPasswordDoesNotMeetRequiremens
	}

	insertedUser, err := s.repo.CreateUser(newUser, password)
	if err != nil {
		return err
	}

	err = s.sendConfirmationRequest(newUser.Email, insertedUser.ConfirmationUUID, serverAddr)
	if err != nil {
		s.repo.DeleteUser(insertedUser.ID)
		return err
	}

	return nil
}

func (s *service) ConfirmAccount(token string) (string, error) {
	return s.repo.SetConfirmationIDToNULL(token)
}

func (s *service) ChangePassword(userid int, oldPassword, newPassword string) error {
	id, err := s.repo.CompareCredentials(userid, oldPassword)
	if err != nil {
		return err
	}

	if id == -1 {
		return core.ErrAccessDenied
	}

	err = s.repo.SetPassword(userid, newPassword)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) sendConfirmationRequest(emailAddress string, confirmationUUID uuid.UUID, baseURL string) error {
	body :=
		"Bitte klicken Sie diesen Link um Ihr Konto zu bestätigen: \r\n" +
			fmt.Sprintf("%s/confirm?token=%s\r\n", baseURL, confirmationUUID.String())
	err := s.mailing.SendEmail(emailAddress, "DevChat-Kontoverwaltung", body)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetAvatar(userID int, pathPrefix string, nodefault bool) (string, time.Time, error) {
	modTime := time.Time{}
	avatarPath := path.Join(pathPrefix, strconv.Itoa(userID))
	stats, err := os.Stat(avatarPath)
	if os.IsNotExist(err) {
		if nodefault {
			return "", modTime, core.ErrRessourceDoesNotExist
		}

		avatarPath = path.Join(pathPrefix, "default.png")
	} else if err != nil {
		return "", modTime, err
	} else {
		modTime = stats.ModTime()
	}

	return avatarPath, modTime, nil
}

// checkPasswordPolicy returns true when the requirements are not met,
// otherwise false.
func checkPasswordPolicy(password string) bool {
	if len(password) < 6 {
		return true
	}
	return false
}
