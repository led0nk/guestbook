package jsondb

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/internal/model"
)

func TestCreateUser(t *testing.T) {
	var testuser = model.User{
		ID:    uuid.New(),
		Name:  "peter müller",
		Email: "peter@müller.de",
	}

	ustore, err := CreateUserStorage("../../testdata/test.json")
	if err != nil {
		t.Errorf("couldn't create userstorage")
	}

	got, err := ustore.CreateUser(&testuser)

	if got == uuid.Nil {
		t.Errorf("expected %v, got %v", testuser.ID, got)
	}

	if err != nil {
		t.Errorf("expected %v, got %v", nil, err)
	}
}

func TestValidateUserInput(t *testing.T) {

	var v url.Values
	v.Add("firstname", "peter")
	v.Add("lastname", "müller")

	got := ValidateUserInput(v)

	if got != nil {
		t.Errorf("expected %v, got %v", nil, got)
	}
}
