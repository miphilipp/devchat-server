package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	core "github.com/miphilipp/devchat-server/internal"
)

func (s *service) SendPasswordResetMail(emailAddress, baseURL, language string) error {

	token, err := s.repo.CreateRecoverID(emailAddress)
	if err != nil {
		return err
	}

	body :=
		"Bitte klicken Sie auf diesen Link um Ihr Konto zu bestätigen: \r\n\r\n" +
			fmt.Sprintf("%s/forgot?token=%s\r\n", baseURL, token) +
			"\r\nFür den Fall, dass Sie diese Mail unerwartet erhalten haben, ignorieren Sie diese Mail bitte."
	err = s.mailing.SendEmail(emailAddress, "DevChat-Passwortwiederherstellung", body)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) ResetPassword(recoveryUUID string, newPassword string) (string, error) {
	uuid, err := uuid.Parse(recoveryUUID)
	if err != nil {
		return "", core.ErrInvalidToken
	}

	if checkPasswordPolicy(newPassword) {
		return "", core.ErrPasswordDoesNotMeetRequiremens
	}

	issueDate, err := s.repo.SelectRecoveryTokenIssueDate(uuid)
	if err != nil {
		return "", err
	}

	if time.Now().Add(-5 * time.Minute).UTC().After(issueDate) {
		return "", core.ErrExpired
	}

	username, err := s.repo.RecoverPassword(uuid, newPassword)
	if err != nil {
		return "", err
	}
	return username, nil
}
