package db

import (
	"github.com/led0nk/guestbook/model"

	"github.com/google/uuid"
)

type Storage interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
	GetEntryByName(string) ([]*model.GuestbookEntry, error)
}
