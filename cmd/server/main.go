package main

import (
	"context"
	"flag"
	"log/slog"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		addr        = flag.String("addr", "localhost:8080", "server port")
		grpcaddr    = flag.String("grpcaddr", "", "grpc address, e.g. localhost:4317")
		dbase       = flag.String("db", "file://testdata", "path to database")
		envStr      = flag.String("env", "testdata/.env", "path to .env-file")
		domain      = flag.String("domain", "127.0.0.1", "given domain for cookies/mail")
		logLevelStr = flag.String("loglevel", "INFO", "define the level for logs")
		bStore      db.GuestBookStore
		uStore      db.UserStore
		tStore      db.TokenStore
	)
	flag.Parse()
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(*logLevelStr))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	if err != nil {
		logger.Error("error parsing loglevel", "loglevel", *logLevelStr, "error", err)
	}
	slog.SetDefault(logger)

	logger.Info("server address", "addr", *addr)
	logger.Info("otlp/grpc", "gprcaddr", *grpcaddr)
	logger.Info("path to data", "db", *dbase)
	logger.Info("path to .env", "env", *envStr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if *grpcaddr != "" {
		//NOTE: grpc configuration
		grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock()}
		conn, err := grpc.NewClient(*grpcaddr, grpcOptions...)
		if err != nil {
			logger.Error("failed to create grpc client", err)
			os.Exit(1)
		}
		defer conn.Close()

		//NOTE: tracing configuration
		oteltraceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			logger.Error("failed to create otlp trace exporter", err)
			os.Exit(1)
		}
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(oteltraceExporter))
		otel.SetTracerProvider(tp)

		//NOTE: metrics configuration
		otelmetricsExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			logger.Error("failed to create otlp metrics exporter", err)
			os.Exit(1)
		}
		mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(otelmetricsExporter)))
		otel.SetMeterProvider(mp)
	}

	//NOTE: load .env file / creates if none provided
	envmap, err := utils.LoadEnv(logger, *envStr)
	if err != nil {
		logger.Error("failed to load .env variables", err)
	}

	tStore, err = token.CreateTokenService(envmap["TOKENSECRET"])
	if err != nil {
		logger.Error("failed to create token service", err)
	}

	u, err := url.Parse(*dbase)
	if err != nil {
		panic(err)
	}
	switch u.Scheme {
	case "file":
		filepath := u.Host + u.Path
		bStore, err = jsondb.CreateBookStorage(filepath + "/entries.json")
		if err != nil {
			logger.Error("couldn't create entry storage", err)
		}

		uStore, err = jsondb.CreateUserStorage(filepath + "/user.json")
		if err != nil {
			logger.Error("couldn't create user storage", err)
		}
	default:
		logger.Error("no database provided", "dbase", u.Scheme)
		os.Exit(1)
	}

	templates := templates.NewTemplateHandler()

	mailer := mailer.NewMailer(
		envmap["EMAIL"],
		envmap["SMTPPW"],
		envmap["HOST"],
		envmap["PORT"])

	server := v1.NewServer(*addr, mailer, *domain, templates, bStore, uStore, tStore)
	server.ServeHTTP()
}
