package database

import (
	"net/http"

	"github.com/led0nk/guestbook/internal/model"

	"github.com/google/uuid"
)

type Database interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
	GetEntryByName(string) ([]*model.GuestbookEntry, error)
	GetEntryByID(uuid.UUID) ([]*model.GuestbookEntry, error)

	CreateUser(*model.User) (uuid.UUID, error)
	GetUserByEmail(string) (*model.User, error)
	GetUserByID(uuid.UUID) (*model.User, error)

	CreateToken(uuid.UUID) (*http.Cookie, error)
	DeleteToken(uuid.UUID) error
	GetTokenValue(*http.Cookie) (uuid.UUID, error)
	Valid(string) (bool, error)
	Refresh(string) (*http.Cookie, error)
}
