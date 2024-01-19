package v1

import db "github.com/led0nk/guestbook/internal/database"

type Server struct {
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}
