package utils

import (
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

func CheckFlag(flag *string, logger zerolog.Logger, fn func(string) (interface{}, error)) interface{} {
	var rStore interface{}
	u, err := url.Parse(*flag)
	if err != nil {
		panic(err)
	}
	logger.Info().Msg(u.String())
	switch u.Scheme {
	case "file":
		logger.Info().Str("opening", u.Host+u.Path).Msg("")
		storage, _ := fn(u.Host + u.Path)
		rStore = storage
	default:
		panic("bad storage")
	}
	return rStore
}

// Create a Random String e.g. for Verification Code
func RandomString(l int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	randomString := make([]rune, l)
	for i := range randomString {
		randomString[i] = chars[rand.Intn(len(chars))]
	}
	return string(randomString)
}

func FormValueBool(s string) bool {
	return s == "true"
}

func Capitalize(s string) string {
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// LoadEnv loads env vars from .env
func LoadEnv(logger zerolog.Logger, path string) (map[string]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0777)
		if err != nil {
			return nil, err
		}
		envMap := make(map[string]string, 0)
		envMap["TOKENSECRET"] = "secret"
		err := godotenv.Write(envMap, path)
		if err != nil {
			return nil, err
		}
		logger.Info().Str("created", path).Msg("")
	}

	envmap, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}

	for k, v := range envmap {
		if v == "" {
			logger.Warn().Str(k, "empty value").Msg("")
		}
	}

	return envmap, nil
}
