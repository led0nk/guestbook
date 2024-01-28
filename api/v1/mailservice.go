package v1

import (
	templates "github.com/led0nk/guestbook/internal"
	"github.com/led0nk/guestbook/internal/model"
)

type Mailerservice interface {
	SendVerMail(*model.User, *templates.TemplateHandler) error
}
