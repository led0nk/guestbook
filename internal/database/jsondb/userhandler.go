package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/cmd/utils"
	"github.com/led0nk/guestbook/internal/model"
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
		return errors.New("file does not exist")
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

// maybe only for expired ExpTime and Reverification
func (u *UserStorage) CreateVerificationCode(userID uuid.UUID) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if userID == uuid.Nil {
		return errors.New("User ID is empty")
	}
	u.user[userID].VerificationCode = utils.RandomString(6)
	u.user[userID].ExpirationTime = time.Now().Add(time.Minute * 5)
	return nil
}

func (u *UserStorage) UpdateUser(user *model.User) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.user[user.ID] = user
	if err := u.writeUserJSON(); err != nil {
		return err
	}
	return nil
}

func (u *UserStorage) GetUserByEmail(email string) (*model.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if email == "" {
		return nil, errors.New("requires an email input")
	}

	users := &model.User{}
	for _, user := range u.user {
		if user.Email == email {
			users = user
		}
	}
	return users, nil
}

func (u *UserStorage) GetUserByID(ID uuid.UUID) (*model.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ID == uuid.Nil {
		return nil, errors.New("UUID empty")
	}

	users := &model.User{}
	for _, user := range u.user {
		if user.ID == ID {
			users = user
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

	_, emailValid := mail.ParseAddress(v.Get("email"))
	if emailValid != nil {
		return errors.New("email is not in correct format, please try again")
	}
	return nil
}

func (u *UserStorage) GetUserByToken(token string) (*model.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	returnvalue := &model.User{}

	return returnvalue, nil
}

func (u *UserStorage) CodeValidation(ID uuid.UUID, code string) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	user, err := u.GetUserByID(ID)
	if err != nil {
		return false, err
	}
	if !time.Now().Before(user.ExpirationTime) {
		u.DeleteUser(ID)
		return false, errors.New("Verification Code expired")
	}
	if user.VerificationCode != code {
		return false, errors.New("Wrong Verification Code")
	}
	user.IsVerified = true
	fmt.Println("test2")
	u.UpdateUser(user)
	fmt.Println("test2")
	return true, nil
}

// delete Entry from storage and write to JSON
func (u *UserStorage) DeleteUser(ID uuid.UUID) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ID == uuid.Nil {
		return errors.New("requires an entryID")
	}
	if _, exists := u.user[ID]; !exists {
		err := errors.New("entry doesn't exist")
		return err
	}

	delete(u.user, ID)

	if err := u.writeUserJSON(); err != nil {
		return err
	}

	return nil
}
