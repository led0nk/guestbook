package token

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("github.com/led0nk/guestbook/token")

type Token struct {
	Token      string
	Expiration time.Time
}

type TokenStorage struct {
	Tokens map[uuid.UUID]*Token
	Secret string
	mu     sync.Mutex
}

func CreateTokenService(secret string) (*TokenStorage, error) {
	tokenService := &TokenStorage{
		Tokens: make(map[uuid.UUID]*Token),
		Secret: secret,
	}
	return tokenService, nil
}

func (t *TokenStorage) CreateToken(ctx context.Context, session string, domain string, ID uuid.UUID, remember bool) (*http.Cookie, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "CreateToken")
	defer span.End()

	span.AddEvent("Lock")
	t.mu.Lock()
	defer span.AddEvent("Unlock")
	defer t.mu.Unlock()
	if ID == uuid.Nil {
		return nil, errors.New("Cannot create Token for empty User ID")
	}

	span.AddEvent("create token")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": ID.String(),
	})
	span.AddEvent("sign token")
	tokenString, _ := token.SignedString([]byte(t.Secret)) //exchange to osenv later
	expiration := time.Now().Add(15 * time.Minute)
	tokenStruct := &Token{
		Token:      tokenString,
		Expiration: expiration,
	}
	if remember {
		tokenStruct.Expiration = time.Now().Add(24 * time.Hour)
	}
	t.Tokens[ID] = tokenStruct

	cookie := http.Cookie{
		Name:     session,
		Value:    tokenString,
		Domain:   domain,
		Path:     "/",
		Expires:  expiration,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	return &cookie, nil
}

func (t *TokenStorage) DeleteToken(ctx context.Context, ID uuid.UUID) error {
	var span trace.Span
	_, span = tracer.Start(ctx, "DeleteToken")
	defer span.End()

	span.AddEvent("Lock")
	t.mu.Lock()
	defer span.AddEvent("Unlock")
	defer t.mu.Unlock()
	if ID == uuid.Nil {
		return errors.New("Cannot delete Token for empty User ID")
	}

	if _, exists := t.Tokens[ID]; !exists {
		return errors.New("there is no token existing for this ID")
	}
	span.AddEvent("delete Token")
	delete(t.Tokens, ID)
	return nil
}

func (t *TokenStorage) GetTokenValue(ctx context.Context, c *http.Cookie) (uuid.UUID, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetTokenValue")
	defer span.End()

	tokenString := c.Value
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(t.Secret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	tokenClaims, _ := token.Claims.(jwt.MapClaims)
	valueString := tokenClaims["id"].(string)
	span.AddEvent("parse to uuid")
	tokenValue, _ := uuid.Parse(valueString)

	return tokenValue, nil

}
func (t *TokenStorage) Valid(ctx context.Context, val string) (bool, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "Valid")
	defer span.End()

	span.AddEvent("range over tokens")
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

func (t *TokenStorage) Refresh(ctx context.Context, val string) (*http.Cookie, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "Refresh")
	defer span.End()

	if val == "" {
		return nil, errors.New("refresh failed, empty value")
	}

	span.AddEvent("set cookie values")
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
