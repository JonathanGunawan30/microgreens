package email

import (
	"notification-service/config"
	"notification-service/internal/core/port"

	"github.com/go-mail/mail"
)

type smtpEmail struct {
	Username string
	Host     string
	Password string
	Port     int
	From     string
	Tls      bool
}

func NewSmtpEmail(cfg *config.Config) port.EmailSender {
	return &smtpEmail{
		Username: cfg.Email.Username,
		Host:     cfg.Email.Host,
		Password: cfg.Email.Password,
		Port:     cfg.Email.Port,
		From:     cfg.Email.From,
		Tls:      cfg.Email.TLS,
	}
}

func (e *smtpEmail) SendEmailNotif(to, subject, body string) error {
	m := mail.NewMessage()
	m.SetHeader("From", e.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	m.SetBody("text/html", body)

	d := mail.NewDialer(e.Host, e.Port, e.Username, e.Password)

	if e.Tls {
		d.StartTLSPolicy = mail.MandatoryStartTLS
	}

	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
