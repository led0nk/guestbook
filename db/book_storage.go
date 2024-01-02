package db

import (
	"guestbook/db/jsondb"

	"github.com/google/uuid"
)

type BookStorage interface {
	CreateEntry(*jsondb.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*jsondb.GuestbookEntry, error)
	DeleteEntry()
}
