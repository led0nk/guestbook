package mailer

import (
	"bytes"
	"net/smtp"

	templates "github.com/led0nk/guestbook/internal"
	"github.com/led0nk/guestbook/internal/model"
)

type Mailer struct {
	Email    string
	Password string
	Host     string
	Port     string
}

func NewMailer(
	email string,
	password string,
	host string,
	port string,
) *Mailer {
	return &Mailer{
		Email:    email,
		Password: password,
		Host:     host,
		Port:     port,
	}
}

type data struct {
	User   *model.User
	Domain string
}

func (m *Mailer) SendVerMail(user *model.User, domain string, tmpl *templates.TemplateHandler) error {
	var body bytes.Buffer

	data := &data{
		User:   user,
		Domain: domain,
	}

	err := tmpl.TmplVerMail.Execute(&body, data)
	if err != nil {
		return err
	}
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
	msg := "Subject: Email Validation" + "\n" + headers + "\n\n" + body.String()
	err = smtp.SendMail(
		m.Host+":"+m.Port,
		smtp.PlainAuth(
			"",
			m.Email,
			m.Password,
			m.Host,
		),
		m.Email,
		[]string{user.Email},
		[]byte(msg),
	)
	if err != nil {
		return err
	}
	return nil
}

func (m *Mailer) SendPWMail(user *model.User, tmpl *templates.TemplateHandler) error {
	var body bytes.Buffer
	err := tmpl.TmplVerMail.Execute(&body, user)
	if err != nil {
		return err
	}
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
	msg := "Subject: Email Validation" + "\n" + headers + "\n\n" + body.String()
	err = smtp.SendMail(
		m.Host+":"+m.Port,
		smtp.PlainAuth(
			"",
			m.Email,
			m.Password,
			m.Host,
		),
		m.Email,
		[]string{user.Email},
		[]byte(msg),
	)
	if err != nil {
		return err
	}
	return nil
}
