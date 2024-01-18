package db

import (
	"net/http"

	"github.com/led0nk/guestbook/model"

	"github.com/google/uuid"
)

type GuestBookStore interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
	GetEntryByName(string) ([]*model.GuestbookEntry, error)
	GetEntryByID(uuid.UUID) ([]*model.GuestbookEntry, error)
}

type UserStore interface {
	CreateUser(*model.User) (uuid.UUID, error)
	GetUserByEmail(string) (*model.User, error)
	GetUserByID(uuid.UUID) (*model.User, error)
}

type TokenStore interface {
	CreateToken(uuid.UUID) (*http.Cookie, error)
	DeleteToken(uuid.UUID) error
	GetTokenValue(*http.Cookie) (uuid.UUID, error)
	Valid(string) (bool, error)
	Refresh(string) (*http.Cookie, error)
}
