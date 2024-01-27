package v1

import (
	templates "github.com/led0nk/guestbook/internal"
	"github.com/led0nk/guestbook/internal/mailer"
	"github.com/led0nk/guestbook/internal/model"
)

type Mailersvc interface {
	NewMailer(string, string, string, string) *mailer.Mailer
	SendVerMail(*model.User, *templates.TemplateHandler) error
}
