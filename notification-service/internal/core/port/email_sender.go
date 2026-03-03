package port

type EmailSender interface {
	SendEmailNotif(to, subject, body string) error
}
