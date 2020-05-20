package mailing

import (
	core "github.com/miphilipp/devchat-server/internal"
	"gopkg.in/gomail.v2"
)

type mailingService struct {
	user        string
	password    string
	senderEmail string
	server      string
	port        uint16
}

func NewService(server string, port uint16, password, user, senderEmail string) core.MailingService {
	return &mailingService{
		user:        user,
		password:    password,
		senderEmail: senderEmail,
		port:        port,
		server:      server,
	}
}

func (m *mailingService) SendEmail(to, subject, body string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.senderEmail)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	d := gomail.NewDialer(m.server, int(m.port), m.user, m.password)
	if err := d.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
