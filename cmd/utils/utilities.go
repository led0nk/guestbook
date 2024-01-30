package utils

import (
	"math/rand"
	"net/url"

	log "github.com/sirupsen/logrus"
)

// protection from nil pointers
func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// TODO
func CheckFlag(flag *string,log *log.Logger, storage any) any {
	path, err := url.Parse(*flag)
	if err != nil {
		panic(err)
	}
	log.Info(path)
	switch path.Scheme {
	case "file":
		log.Info("opening: ", path.Host+path.Path)

	default:

	}
	return nil
}

// Create a Random String e.g. for Verification Code
func RandomString(l int) string{
  var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

  randomString := make([]rune, l)
  for i := range randomString {
    randomString[i] = chars[rand.Intn(len(chars))]
  }
  return string(randomString)
}
