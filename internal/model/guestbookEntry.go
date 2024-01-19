package model

import (
	"github.com/google/uuid"
)

type GuestbookEntry struct {
	ID        uuid.UUID `json:"id" form:"-"`
	Name      string    `json:"name"`
	Message   string    `json:"message"`
	CreatedAt string    `json:"created_at"`
	UserID    uuid.UUID `json:"userid" form:"-"`
}
