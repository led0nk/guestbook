package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/model"
)

type UserStorage struct {
	filename string
	user     map[uuid.UUID]*model.User
	mu       sync.Mutex
}

func CreateUserStorage(filename string) (*UserStorage, error) {
	storage := &UserStorage{
		filename: filename,
		user:     make(map[uuid.UUID]*model.User),
	}
	if err := storage.readUserJSON(); err != nil {
		return nil, err
	}
	return storage, nil
}

// write JSON data into readable format in file = filename
func (u *UserStorage) writeUserJSON() error {

	as_json, err := json.MarshalIndent(u.user, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(u.filename, as_json, 0644)
	if err != nil {
		return err
	}
	return nil
}

// read JSON data from file = filename
func (u *UserStorage) readUserJSON() error {
	if _, err := os.Stat(u.filename); os.IsNotExist(err) {
		fmt.Println("file does not exist", err)
		return nil
	}
	data, err := os.ReadFile(u.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &u.user)
}

func (u *UserStorage) CreateUser(user *model.User) (uuid.UUID, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	u.user[user.ID] = user

	if err := u.writeUserJSON(); err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

func (u *UserStorage) GetUserByEmail(email string) ([]*model.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if email == "" {
		return nil, errors.New("requires an email input")
	}

	users := []*model.User{}
	for _, user := range u.user {
		if user.Email == email {
			users = append(users, user)
		}
	}

	return users, nil
}

func ValidateUserInput(v url.Values) error {

	if v.Get("firstname") == "" || v.Get("lastname") == "" {
		return errors.New("fields cannot be empty")
	}
	if v["password"][0] != v["password"][1] {
		return errors.New("password doesn't match, please try again")
	}
	if len(v["password"][0]) > 72 || len(v["password"][1]) > 72 {
		return errors.New("password is too long, only 72 characters allowed")
	}
	if len(v["password"][0]) < 8 || len(v["password"][1]) < 8 {
		return errors.New("password is too short, should be at least 8 characters long")
	}
	/*if strings.ContainsAny(v["password"][0], "[0-9]") == false {
		return errors.New("password does not contain any numbers, please correct")
	}*/
	_, emailValid := mail.ParseAddress(v.Get("email"))
	if emailValid != nil {
		return errors.New("email is not in correct format, please try again")
	}
	return nil
}
