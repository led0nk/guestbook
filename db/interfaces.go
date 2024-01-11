package db

import (
	"net/url"

	"github.com/led0nk/guestbook/model"

	"github.com/google/uuid"
)

type GuestBookStorage interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
	GetEntryByName(string) ([]*model.GuestbookEntry, error)
}

type UserStorage interface {
	CreateUser(*model.User) (uuid.UUID, error)
	GetUserByEmail(string) ([]*model.User, error)
	GetHash(string) ([]byte, error)
	ValidateUserInput(url.Values) error
}
