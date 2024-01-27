package model

import (
	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID         `json:"id" form:"-"`
	Email      string            `json:"email"`
	Name       string            `json:"name"`
	Password   []byte            `json:"password"`
	Entry      []*GuestbookEntry `json:"entry"`
	IsAdmin    bool              `json:"isadmin"`
	IsVerified bool              `json:"isverified"`
}
