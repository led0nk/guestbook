package db

import (
	"context"
	"net/http"

	"github.com/led0nk/guestbook/internal/model"

	"github.com/google/uuid"
)

type GuestBookStore interface {
	CreateEntry(context.Context, *model.GuestbookEntry) (uuid.UUID, error)
	ListEntries(context.Context) ([]*model.GuestbookEntry, error)
	DeleteEntry(context.Context, uuid.UUID) error
	GetEntryByName(context.Context, string) ([]*model.GuestbookEntry, error)
	GetEntryByID(context.Context, uuid.UUID) ([]*model.GuestbookEntry, error)
	GetEntryBySnippet(context.Context, string) ([]*model.GuestbookEntry, error)
}

type UserStore interface {
	CreateUser(context.Context, *model.User) (uuid.UUID, error)
	GetUserByEmail(context.Context, string) (*model.User, error)
	GetUserByID(context.Context, uuid.UUID) (*model.User, error)
	UpdateUser(context.Context, *model.User) error
	CreateVerificationCode(context.Context, uuid.UUID) error
	CodeValidation(context.Context, uuid.UUID, string) (bool, error)
	ListUser(context.Context) ([]*model.User, error)
	DeleteUser(context.Context, uuid.UUID) error
}

type TokenStore interface {
	CreateToken(context.Context, string, string, uuid.UUID, bool) (*http.Cookie, error)
	DeleteToken(context.Context, uuid.UUID) error
	GetTokenValue(context.Context, *http.Cookie) (uuid.UUID, error)
	Valid(context.Context, string) (bool, error)
	Refresh(context.Context, string) (*http.Cookie, error)
}
