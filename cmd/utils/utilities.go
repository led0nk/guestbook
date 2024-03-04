package utils

import (
	"math/rand"
	"net/url"
	"os"
	"unicode"

	"regexp"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// protection from nil pointers
func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// TODO
func CheckFlag(flag *string, log zerolog.Logger, storage any) any {
	path, err := url.Parse(*flag)
	if err != nil {
		panic(err)
	}
	log.Info().Msg(path.String())
	switch path.Scheme {
	case "file":
		log.Info().Str("opening: ", path.Host+path.Path).Msg("")

	default:

	}
	return nil
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
func LoadEnv(logger zerolog.Logger) {
	re := regexp.MustCompile(`^(.*` + "guestbook" + `)`)
	cwd, _ := os.Getwd()
	rootPath := re.Find([]byte(cwd))

	err := godotenv.Load(string(rootPath) + `/testdata` + `/.env`)
	if err != nil {
		logger.Fatal().Err(err).Msg(cwd)

		os.Exit(-1)
	}
}
