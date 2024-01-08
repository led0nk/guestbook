package model

import (
	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID      `json:"id" form:"-"`
	Name     string         `json:"name"`
	Password string         `json:"password"`
	Entry    GuestbookEntry `json:"entry"`
}
