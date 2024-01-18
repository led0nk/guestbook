package token

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Token struct {
	Token      string
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

func (t *TokenStorage) CreateToken(ID uuid.UUID) (*http.Cookie, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ID == uuid.Nil {
		return nil, errors.New("Cannot create Token for empty User ID")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": ID.String(),
	})
	tokenString, _ := token.SignedString([]byte("secret")) //exchange to osenv later
	expiration := time.Now().Add(15 * time.Minute)
	tokenStruct := &Token{
		Token:      tokenString,
		Expiration: expiration,
	}

	t.Tokens[ID] = tokenStruct

	cookie := http.Cookie{
		Name:    "session",
		Value:   tokenString,
		Path:    "/",
		Expires: expiration,
		//MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return &cookie, nil
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
func (t *TokenStorage) Valid(val string) (bool, error) {

	for _, token := range t.Tokens {

		if token.Token == val {
			if token.Expiration.Before(time.Now()) {
				return false, errors.New("Token expired")
			}
			return true, nil
		}
	}

	return false, errors.New("Token was not found")
}

func (t *TokenStorage) Refresh(val string) (*http.Cookie, error) {

	if val == "" {
		return nil, errors.New("refresh failed, empty value")
	}

	cookie := http.Cookie{
		Name:    "session",
		Value:   val,
		Path:    "/",
		Expires: time.Now().Add(15 * time.Minute),
		//MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	return &cookie, nil
}
