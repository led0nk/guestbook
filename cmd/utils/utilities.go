package utils

import (
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
func checkFlag(flag *string,log *log.Logger, storage any) any {
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
