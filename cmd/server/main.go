package main

import (
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
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
		addr = flag.String("addr",
			"localhost:8080",
			"server port")
		entryStr = flag.String("entrydata",
			"file://../../testdata/entries.json",
			"link/path to entry-database")
		userStr = flag.String("userdata",
			"file://user.json",
			"link to user-database")
		envStr = flag.String("envvar's",
			"testdata/.env",
			"path to .env-file")
		bStore db.GuestBookStore
		uStore db.UserStore
		tStore db.TokenStore
	)
	flag.Parse()
	bStore = utils.CheckFlag(entryStr, logger,
		func(string) (interface{}, error) {
			result, err := jsondb.CreateBookStorage(utils.DerefString(entryStr))
			if err != nil {
				panic(err)
			}
			return result, nil
		}).(*jsondb.BookStorage)

	uStore = utils.CheckFlag(entryStr, logger,
		func(string) (interface{}, error) {
			result, err := jsondb.CreateUserStorage(utils.DerefString(userStr))
			if err != nil {
				panic(err)
			}
			return result, nil
		}).(*jsondb.UserStorage)

	err := godotenv.Load(utils.DerefString(envStr))
	if err != nil {
		logger.Error().Err(err).Msg("")
		panic("bad mailer env")
	}

	//in memory
	tokenStorage, err := token.CreateTokenService(os.Getenv("TOKENSECRET"))
	if err != nil {
		logger.Error().Err(err).Msg("")
	}
	tStore = tokenStorage

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
