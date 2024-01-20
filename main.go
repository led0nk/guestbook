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

type Store struct {
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewHandler(bookstore db.GuestBookStore, userstore db.UserStore, tokenstore db.TokenStore) *Store {
	return &Store{
		bookstore:  bookstore,
		userstore:  userstore,
		tokenstore: tokenstore,
	}
}

func main() {

	logger := logrus.New()
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)

	var (
		addr     = flag.String("addr", "localhost:8080", "server port")
		entryStr = flag.String("entrydata", "file://internal/database/jsondb/entries.json", "link to entry-database")
		//userStr  = flag.String("userdata", "file://user.json", "link to user-database")
	)
	flag.Parse()
	u, err := url.Parse(*entryStr)
	if err != nil {
		panic(err)
	}
	log.Info(u)
	var bStore db.GuestBookStore
	var uStore db.UserStore
	var tStore db.TokenStore
	address := DerefString(addr)
	//bookString := DerefString(entryStr)
	switch u.Scheme {
	case "file":
		log.Info("opening: ", u.Host+u.Path)
		bookStorage, _ := jsondb.CreateBookStorage(u.Host + u.Path)
		userStorage, _ := jsondb.CreateUserStorage("./user.json")
		tokenStorage, _ := token.CreateTokenService()
		bStore = bookStorage
		uStore = userStorage
		tStore = tokenStorage

	default:
		panic("bad storage")
	}

	server := v1.NewServer(address, logger, bStore, uStore, tStore, middleware.Logger(), middleware.Auth(tStore))
	server.ServeHTTP()
}

func DerefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}
