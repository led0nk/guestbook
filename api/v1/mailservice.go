package v1

import (
	templates "github.com/led0nk/guestbook/internal"
	"github.com/led0nk/guestbook/internal/model"
)

// interface for Mailerservice for Verification-Mail and Reset-PW-Mail
type Mailerservice interface {
	SendVerMail(*model.User, string, *templates.TemplateHandler) error
	SendPWMail(*model.User, *templates.TemplateHandler) error
}
