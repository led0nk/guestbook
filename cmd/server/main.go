package main

import (
	"flag"
	"net/url"
	"os"
	"time"

	v1 "github.com/led0nk/guestbook/api/v1"
	"github.com/led0nk/guestbook/cmd/utils"
	templates "github.com/led0nk/guestbook/internal"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/mailer"
	"github.com/led0nk/guestbook/token"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	var (
		addr     = flag.String("addr", "localhost:8080", "server port")
		entryStr = flag.String("entrydata", "file://../../testdata/entries.json", "link to entry-database")
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
	logger.Info().Msg(u.String())
	switch u.Scheme {
	case "file":
		logger.Info().Msg("opening: " + u.Host + u.Path)
		bookStorage, _ := jsondb.CreateBookStorage(u.Host + u.Path)
		userStorage, _ := jsondb.CreateUserStorage("../../testdata/user.json")
		bStore = bookStorage
		uStore = userStorage

	default:
		panic("bad storage")
	}
	//in memory
	tokenStorage, _ := token.CreateTokenService(os.Getenv("TOKENSECRET"))

	tStore = tokenStorage

	//	err = godotenv.Load("../../testdata/.env")
	//	if err != nil {
	//		panic("bad mailer env")
	//	}
	//protect from nil pointer
	address := utils.DerefString(addr)

	//create templatehandler
	templates := templates.NewTemplateHandler()
	//create mailerservice
	utils.LoadEnv(logger)
	mailer := mailer.NewMailer(
		os.Getenv("EMAIL"),
		os.Getenv("SMTPPW"),
		os.Getenv("HOST"),
		os.Getenv("PORT"))
	//create Server
	server := v1.NewServer(address, mailer, templates, logger, bStore, uStore, tStore)
	server.ServeHTTP()
}
