package db

import (
	"guestbook/model"

	"github.com/google/uuid"
)

type Storage interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
	Submit()
}
