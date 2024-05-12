package utils

import (
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"unicode"

	"github.com/joho/godotenv"
)

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
func LoadEnv(logger *slog.Logger, path string) (map[string]string, error) {
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
		logger.Info("created .env at", "path", path)
	}

	envmap, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}

	for k, v := range envmap {
		if v == "" {
			logger.Warn("empty value in .env", "value", k)
		}
	}

	return envmap, nil
}
