package db

import (
	"github.com/led0nk/guestbook/db/jsondb"

	"github.com/google/uuid"
)

type BookStorage interface {
	CreateEntry(*jsondb.GuestbookEntry) uuid.UUID
	ListEntry()
	DeleteEntry()
}
