package token

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Token struct {
	Token      *jwt.Token
	Expiration time.Time
}

type TokenStorage struct {
	Token map[string]*Token
	mu    sync.Mutex
}

func CreateTokenStorage() (*TokenStorage, error) {
	tokenStorage := &TokenStorage{
		Token: make(map[string]*Token),
	}
	return tokenStorage, nil
}

func (t *TokenStorage) CreateToken(ID string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ID == "" {
		return "", errors.New("Cannot create Token for empty User ID")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": ID,
	})
	tokenString, _ := token.SignedString([]byte("secret")) //exchange to osenv later

	return tokenString, nil
}

func (t *TokenStorage) DeleteToken(ID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ID == "" {
		return errors.New("Cannot delete Token for empty User ID")
	}

	if _, exists := t.Token[ID]; !exists {
		return errors.New("there is no token existing for this ID")
	}
	delete(t.Token, ID)
	return nil
}
