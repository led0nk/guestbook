package token

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenService interface {
	CreateToken(uuid.UUID) (string, time.Time, error)
	DeleteToken(uuid.UUID) error
	GetTokenValue(*http.Cookie) (uuid.UUID, error)
	Valid(string, time.Time) (bool, time.Time, error)
}

type Token struct {
	Token      *jwt.Token
	Expiration time.Time
}

type TokenStorage struct {
	Tokens map[uuid.UUID]*Token
	mu     sync.Mutex
}

func CreateTokenService() (*TokenStorage, error) {
	tokenService := &TokenStorage{
		Tokens: make(map[uuid.UUID]*Token),
	}
	return tokenService, nil
}

func (t *TokenStorage) CreateToken(ID uuid.UUID) (string, time.Time, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ID == uuid.Nil {
		return "", time.Now(), errors.New("Cannot create Token for empty User ID")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": ID.String(),
	})
	tokenString, _ := token.SignedString([]byte("secret")) //exchange to osenv later
	expiration := time.Now().Add(15 * time.Minute)
	tokenStruct := &Token{
		Token:      token,
		Expiration: expiration,
	}

	t.Tokens[ID] = tokenStruct

	return tokenString, expiration, nil
}

func (t *TokenStorage) DeleteToken(ID uuid.UUID) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ID == uuid.Nil {
		return errors.New("Cannot delete Token for empty User ID")
	}

	if _, exists := t.Tokens[ID]; !exists {
		return errors.New("there is no token existing for this ID")
	}
	delete(t.Tokens, ID)
	return nil
}

func (t *TokenStorage) GetTokenValue(c *http.Cookie) (uuid.UUID, error) {
	tokenString := c.Value
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	tokenClaims, _ := token.Claims.(jwt.MapClaims)
	valueString := tokenClaims["id"].(string)
	tokenValue, _ := uuid.Parse(valueString)

	return tokenValue, nil

}
func (t *TokenStorage) Valid(val string, exp time.Time) (bool, time.Time, error) {

	claims := jwt.MapClaims{}
	realtoken, err := jwt.ParseWithClaims(val, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return false, time.Time{}, err
	}

	for _, token := range t.Tokens {
		if token.Token == realtoken {
			if token.Expiration.Before(exp) {
				return false, time.Time{}, errors.New("token expiration timers are different, ALARM!")
			}
			if exp.After(time.Now()) {
				exp = time.Now().Add(15 * time.Minute)
			}
			return true, exp, nil
		}
	}

	return false, time.Time{}, errors.New("Token was not found")
}
