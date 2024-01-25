package main

import (
	"flag"
	"net/url"

	v1 "github.com/led0nk/guestbook/api/v1"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/middleware"
	"github.com/led0nk/guestbook/token"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func main() {

	logger := logrus.New()
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)

	var (
		addr     = flag.String("addr", "localhost:8080", "server port")
		entryStr = flag.String("entrydata", "file://../../internal/database/jsondb/entries.json", "link to entry-database")
		//userStr  = flag.String("userdata", "file://user.json", "link to user-database")
		bStore db.GuestBookStore
		uStore db.UserStore
		tStore db.TokenStore
	)

	flag.Parse()
	u, err := url.Parse(*entryStr)
	if err != nil {
		panic(err)
	}
	log.Info(u)
	switch u.Scheme {
	case "file":
		log.Info("opening: ", u.Host+u.Path)
		bookStorage, _ := jsondb.CreateBookStorage(u.Host + u.Path)
		userStorage, _ := jsondb.CreateUserStorage("../../internal/database/jsondb/user.json")
		bStore = bookStorage
		uStore = userStorage

	default:
		panic("bad storage")
	}
	//in memory
	tokenStorage, _ := token.CreateTokenService()
	tStore = tokenStorage

	//protect from nil pointer
	address := DerefString(addr)

	server := v1.NewServer(address, logger, bStore, uStore, tStore, middleware.Logger(), middleware.Auth(tStore))
	server.ServeHTTP()
}

func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func checkFlag(flag *string, storage any) any {
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