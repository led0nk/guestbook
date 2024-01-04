package db

import (
	"guestbook/model"

	"github.com/google/uuid"
)

type BookStorage interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
}
