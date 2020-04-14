package mailing

import (
	"fmt"
	//"time"
	"net/smtp"

	core "github.com/miphilipp/devchat-server/internal"
)

type mailingService struct {
	auth 		smtp.Auth
	senderEmail string
	server 		string
	port 		uint16
}

func NewService(server string, port uint16, password, user, senderEmail string) core.MailingService {
	auth := smtp.PlainAuth(
		"",
		user,
		password,
		server,
	)
	return &mailingService {
		auth: auth,
		senderEmail: senderEmail,
		port: port,
		server: server,
	}
}

func (m *mailingService) SendEmail(to, subject, body string) error {
	
	msg := fmt.Sprintf(
				"To: %s\r\n" +
				"Subject: %s\r\n" +
				"\r\n" +
				"%s\r\n", to, subject, body)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", m.server, m.port),
		m.auth,
		m.senderEmail,
		[]string{to},
		[]byte(msg),
	)
	if err != nil {
		return err
	}

	return nil
}