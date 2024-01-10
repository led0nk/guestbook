package model

import (
	"net/url"
	"reflect"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID `json:"id" form:"-"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Password []byte    `json:"password"`
	//Entry    GuestbookEntry `json:"entry"`
}

func (u *User) Parse(input url.Values) {
	userType := reflect.TypeOf(*u)

}
