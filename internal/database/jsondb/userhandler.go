package jsondb

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/cmd/utils"
	"github.com/led0nk/guestbook/internal/model"
	"go.opentelemetry.io/otel/trace"
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
		err = os.MkdirAll(filepath.Dir(u.filename), 0777)
		if err != nil {
			return err
		}
		err = u.writeUserJSON()
		if err != nil {
			return err
		}
	}
	data, err := os.ReadFile(u.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &u.user)
}

func (u *UserStorage) CreateUser(ctx context.Context, user *model.User) (uuid.UUID, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "CreateUser")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	span.AddEvent("Check for Email")
	for _, userexist := range u.user {
		if userexist.Email == user.Email {
			return uuid.Nil, errors.New("email cannot be used more than once")
		}
	}

	u.user[user.ID] = user
	if err := u.writeUserJSON(); err != nil {
		return uuid.Nil, err
	}

	return user.ID, nil
}

func (u *UserStorage) ListUser(ctx context.Context) ([]*model.User, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "ListUser")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()

	userlist := make([]*model.User, 0, len(u.user))
	for _, user := range u.user {
		userlist = append(userlist, user)
	}
	sort.Slice(userlist, func(i, j int) bool { return userlist[i].Name > userlist[j].Name })
	return userlist, nil
}

// maybe only for expired ExpTime and Reverification
func (u *UserStorage) CreateVerificationCode(ctx context.Context, userID uuid.UUID) error {
	var span trace.Span
	_, span = tracer.Start(ctx, "CreateVerificationCode")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()

	if userID == uuid.Nil {
		return errors.New("User ID is empty")
	}
	u.user[userID].VerificationCode = utils.RandomString(6)
	u.user[userID].ExpirationTime = time.Now().Add(time.Minute * 5)
	return nil
}

func (u *UserStorage) UpdateUser(ctx context.Context, user *model.User) error {
	var span trace.Span
	_, span = tracer.Start(ctx, "UpdateUser")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()

	u.user[user.ID] = user
	if err := u.writeUserJSON(); err != nil {
		return err
	}
	return nil
}

func (u *UserStorage) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetUserByEmail")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()
	if email == "" {
		return nil, errors.New("requires an email input")
	}

	users := &model.User{}
	span.AddEvent("range over user")
	for _, user := range u.user {
		if user.Email == email {
			users = user
		}
	}
	return users, nil
}

func (u *UserStorage) GetUserByID(ctx context.Context, ID uuid.UUID) (*model.User, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetUserByID")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()
	if ID == uuid.Nil {
		return nil, errors.New("UUID empty")
	}
	users := &model.User{}
	span.AddEvent("range over user")
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
	if strings.ContainsAny(v.Get("firstname"), "0123456789") || strings.ContainsAny(v.Get("lastname"), "01234567890") {
		return errors.New("no numbers allowed")
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

func (u *UserStorage) CodeValidation(ctx context.Context, ID uuid.UUID, code string) (bool, error) {
	var span trace.Span
	ctx, span = tracer.Start(ctx, "CodeValidation")
	defer span.End()

	span.AddEvent("get user")
	user, err := u.GetUserByID(ctx, ID)
	if err != nil {
		return false, err
	}
	if !time.Now().Before(user.ExpirationTime) {
		err := u.DeleteUser(ctx, ID)
		if err != nil {
			return false, err
		}
		return false, errors.New("Verification Code expired")
	}
	if user.VerificationCode != code {
		return false, errors.New("Wrong Verification Code")
	}
	user.IsVerified = true
	span.AddEvent("update user")
	err = u.UpdateUser(ctx, user)
	if err != nil {
		return false, err
	}
	return true, nil
}

// delete Entry from storage and write to JSON
func (u *UserStorage) DeleteUser(ctx context.Context, ID uuid.UUID) error {
	var span trace.Span
	_, span = tracer.Start(ctx, "DeleteUser")
	defer span.End()

	span.AddEvent("Lock")
	u.mu.Lock()
	defer span.AddEvent("Unlock")
	defer u.mu.Unlock()
	if ID == uuid.Nil {
		return errors.New("requires an userID")
	}
	if _, exists := u.user[ID]; !exists {
		err := errors.New("user doesn't exist")
		return err
	}

	delete(u.user, ID)

	span.AddEvent("delete user from json")
	if err := u.writeUserJSON(); err != nil {
		return err
	}

	return nil
}
