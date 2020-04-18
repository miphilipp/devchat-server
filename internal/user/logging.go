package user

import (
	"time"

	"github.com/go-kit/kit/log"

	core "github.com/miphilipp/devchat-server/internal"
)

type loggingService struct {
	logger  log.Logger
	next    Service
	verbose bool
}

func NewLoggingService(logger log.Logger, s Service, verbose bool) Service {
	return &loggingService{logger, s, verbose}
}

func (s *loggingService) AuthenticateUser(username string, password string) (userid int, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "AuthenticateUser",
				"username", username,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.AuthenticateUser(username, password)
}

func (s *loggingService) SearchUsers(prefix string) (users []core.User, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "SearchUsers",
				"prefix", prefix,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.SearchUsers(prefix)
}

func (s *loggingService) ResetPassword(recoveryUUID string, newPassword string) (username string, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ResetPassword",
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ResetPassword(recoveryUUID, newPassword)
}

func (s *loggingService) GetUserForID(id int) (user core.User, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "GetUserForID",
				"userID", id,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.GetUserForID(id)
}

func (s *loggingService) GetUserForName(name string) (user core.User, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "GetUserForName",
				"name", name,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.GetUserForName(name)
}

func (s *loggingService) DeleteAccount(userID int) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "DeleteAccount",
				"userID", userID,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.DeleteAccount(userID)
}

func (s *loggingService) CreateAccount(newUser core.User, password string, serverAddr string) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "CreateAccount",
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.CreateAccount(newUser, password, serverAddr)
}

func (s *loggingService) ConfirmAccount(token string) (username string, err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ConfirmAccount",
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ConfirmAccount(token)
}

func (s *loggingService) SendPasswordResetMail(emailAddress, baseURL, language string) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "SendPasswordResetMail",
				"E-Mail", emailAddress,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.SendPasswordResetMail(emailAddress, baseURL, language)
}

func (s *loggingService) ChangeOnlineState(userCtx int, state bool) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ChangeOnlineState",
				"Context", userCtx,
				"State", state,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ChangeOnlineState(userCtx, state)
}

func (s *loggingService) ChangePassword(userid int, oldPassword string, newPassword string) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "ChangePassword",
				"userID", userid,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.ChangePassword(userid, oldPassword, newPassword)
}

func (s *loggingService) SaveAvatar(filePath string, fileType string, buffer []byte) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "SaveAvatar",
				"path", filePath,
				"type", fileType,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.SaveAvatar(filePath, fileType, buffer)
}

func (s *loggingService) DeleteAvatar(filePath string) (err error) {
	defer func(begin time.Time) {
		if err != nil || s.verbose {
			s.logger.Log(
				"Use-Case", "DeleteAvatar",
				"path", filePath,
				"took", time.Since(begin),
				"err", err)
		}
	}(time.Now())
	return s.next.DeleteAvatar(filePath)
}
