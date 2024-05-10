package main

import (
	"context"
	"errors"
	"flag"
	"net/url"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	var (
		addr     = flag.String("addr", "localhost:8080", "server port")
		grpcaddr = flag.String("grpcaddr", "localhost:4317", "grpc address")
		dbase    = flag.String("db", "file://testdata", "path to database")
		envStr   = flag.String("envvar's", "testdata/.env", "path to .env-file")
		domain   = flag.String("domain", "127.0.0.1", "given domain for cookies/mail")
		bStore   db.GuestBookStore
		uStore   db.UserStore
		tStore   db.TokenStore
	)
	flag.Parse()

	u, err := url.Parse(*dbase)
	if err != nil {
		panic(err)
	}
	switch u.Scheme {
	case "file":
		filepath := u.Host + u.Path
		bStore, err = jsondb.CreateBookStorage(filepath + "/entries.json")
		if err != nil {
			logger.Error().Err(errors.New("db")).Msg(err.Error())
		}

		uStore, err = jsondb.CreateUserStorage(filepath + "/user.json")
		if err != nil {
			logger.Error().Err(errors.New("db")).Msg(err.Error())
		}
	default:
		logger.Error().Err(errors.New("db")).Msg("no database provided")
		os.Exit(1)
	}

	err = godotenv.Load(*envStr)
	if err != nil {
		logger.Error().Err(err).Msg("")
		panic("bad mailer env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	//NOTE: grpc configuration
	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock()}
	conn, err := grpc.NewClient(*grpcaddr, grpcOptions...)
	if err != nil {
		logger.Error().Err(err).Msg("")
		os.Exit(1)
	}
	defer conn.Close()

	//NOTE: tracing configuration
	oteltraceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		logger.Error().Err(err).Msg("")
		os.Exit(1)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(oteltraceExporter))
	otel.SetTracerProvider(tp)

	//NOTE: metrics configuration
	otelmetricsExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		logger.Error().Err(err).Msg("")
		os.Exit(1)
	}
	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(otelmetricsExporter)))
	otel.SetMeterProvider(mp)

	//in memory
	tokenStorage, err := token.CreateTokenService(os.Getenv("TOKENSECRET"))
	if err != nil {
		logger.Error().Err(err).Msg("")
	}
	tStore = tokenStorage

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
	server := v1.NewServer(*addr, mailer, *domain, templates, logger, bStore, uStore, tStore)
	server.ServeHTTP()
}
