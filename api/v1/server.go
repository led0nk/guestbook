package v1

import (
	"net/http"

	"github.com/gorilla/mux"
	db "github.com/led0nk/guestbook/internal/database"
)

type Server struct {
	mw         []mux.MiddlewareFunc
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewServer(
	middleware mux.MiddlewareFunc,
	bStore db.GuestBookStore,
	uStore db.UserStore,
	tStore db.TokenStore,
) http.Handler {
	return &Server{
		mw:         middleware,
		bookstore:  bStore,
		userstore:  uStore,
		tokenstore: tStore,
	}
}
