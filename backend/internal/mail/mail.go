package mail

import (
	"fmt"
	"net/smtp"
	"strings"

	"health-checkup/backend/internal/config"
)

type Sender struct {
	config config.Config
}

func NewSender(cfg config.Config) Sender {
	return Sender{config: cfg}
}

func (s Sender) Enabled() bool {
	return s.config.SMTPUser != "" && s.config.SMTPPass != ""
}

func (s Sender) Send(to, subject, body string) error {
	if !s.Enabled() {
		return fmt.Errorf("smtp is not configured")
	}
	host := s.config.SMTPHost
	addr := host + ":" + s.config.SMTPPort
	auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPass, host)
	message := strings.Join([]string{
		"From: " + s.config.SMTPUser,
		"To: " + to,
		"Subject: " + subject,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, s.config.SMTPUser, []string{to}, []byte(message))
}
