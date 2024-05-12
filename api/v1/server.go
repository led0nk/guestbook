package v1

import (
	"html"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	templates "github.com/led0nk/guestbook/internal"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/middleware"
	"github.com/led0nk/guestbook/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sloghttp "github.com/samber/slog-http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("github.com/led0nk/guestbook/api/v1")
var meter = otel.GetMeterProvider().Meter("github.com/led0nk/guestbook/api/v1")

type Server struct {
	addr       string
	mailer     Mailerservice
	domain     string
	templates  *templates.TemplateHandler
	log        *slog.Logger
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewServer(
	address string,
	mailer Mailerservice,
	domain string,
	templates *templates.TemplateHandler,
	bStore db.GuestBookStore,
	uStore db.UserStore,
	tStore db.TokenStore,
) *Server {
	return &Server{
		addr:       address,
		mailer:     mailer,
		domain:     domain,
		templates:  templates,
		log:        slog.Default().WithGroup("http"),
		bookstore:  bStore,
		userstore:  uStore,
		tokenstore: tStore,
	}
}

func (s *Server) ServeHTTP() {
	r := http.NewServeMux()

	otelmw := otelhttp.NewMiddleware("guestbook")
	authmw := middleware.Auth(s.tokenstore, s.log)
	adminmw := middleware.AdminAuth(s.tokenstore, s.userstore, s.log)
	slogmw := sloghttp.NewWithConfig(
		s.log, sloghttp.Config{
			DefaultLevel:     slog.LevelInfo,
			ClientErrorLevel: slog.LevelWarn,
			ServerErrorLevel: slog.LevelError,
			WithUserAgent:    true,
		},
	)
	traceAttrmw := middleware.SlogAddTraceAttributes()

	r.Handle("GET /", http.HandlerFunc(s.handlePage))
	//NOTE: register /metrics
	r.Handle("GET /metrics", promhttp.Handler())
	r.Handle("GET /login", http.HandlerFunc(s.loginHandler))
	r.Handle("POST /login", http.HandlerFunc(s.loginAuth))
	r.Handle("GET /logout", http.HandlerFunc(s.logoutAuth))
	r.Handle("GET /signup", http.HandlerFunc(s.signupHandler))
	r.Handle("POST /signup", http.HandlerFunc(s.signupAuth))
	r.Handle("GET /forgot-pw", http.HandlerFunc(s.forgotHandler))
	r.Handle("POST /forgot-pw", http.HandlerFunc(s.forgotPW))

	r.Handle("GET /user/verify", authmw(http.HandlerFunc(s.verifyHandler)))
	r.Handle("POST /user/verify", authmw(http.HandlerFunc(s.verifyAuth)))
	r.Handle("GET /user/dashboard", authmw(http.HandlerFunc(s.dashboardHandler)))
	r.Handle("POST /user/dashboard/{ID}", authmw(http.HandlerFunc(s.changeUserData)))
	r.Handle("PUT /user/dashboard/{ID}", authmw(http.HandlerFunc(s.submitUserData)))
	r.Handle("GET /user/create", authmw(http.HandlerFunc(s.createHandler)))
	r.Handle("GET /user/search", authmw(http.HandlerFunc(s.searchHandler)))
	r.Handle("GET /user/search/", authmw(http.HandlerFunc(s.search)))
	r.Handle("POST /user/create", authmw(http.HandlerFunc(s.createEntry)))
	r.Handle("PUT /user/dashboard/{ID}/password-reset", authmw(http.HandlerFunc(s.passwordReset)))

	r.Handle("GET /admin/dashboard", adminmw(http.HandlerFunc(s.adminHandler)))
	r.Handle("DELETE /admin/dashboard/{ID}", adminmw(http.HandlerFunc(s.deleteUser)))
	r.Handle("POST /admin/dashboard/{ID}", adminmw(http.HandlerFunc(s.updateUser)))
	r.Handle("PUT /admin/dashboard/{ID}", adminmw(http.HandlerFunc(s.saveUser)))
	r.Handle("PUT /admin/dashboard/{ID}/verify", adminmw(http.HandlerFunc(s.resendVer)))
	r.Handle("PUT /admin/dashboard/{ID}/password-reset", adminmw(http.HandlerFunc(s.passwordReset)))

	s.log.Info("listening to", "addr", s.addr)

	srv := &http.Server{
		Addr:    s.addr,
		Handler: slogmw(traceAttrmw(otelmw(r))),
	}
	err := srv.ListenAndServe()
	if err != nil {
		s.log.Error("error during listen and server", err)
		os.Exit(1)
	}
}

// hands over Entries to Handler and prints them out in template
func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.ListEntries")
	defer span.End()

	var histogram metric.Float64Histogram
	histogram, err := meter.Float64Histogram(
		"server.handlePage",
		metric.WithDescription("Duration of handlePage"),
		metric.WithUnit("s"),
	)
	if err != nil {
		s.log.ErrorContext(ctx, "metrics", err)
	}
	start := time.Now()

	entries, err := s.bookstore.ListEntries(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to list entries", err)
	}
	err = s.templates.TmplHome.Execute(w, &entries)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
	duration := time.Since(start)
	defer histogram.Record(r.Context(), duration.Seconds())
}

// searches with livesearch (htmx)
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.searchHandler")
	defer span.End()

	entries, err := s.bookstore.ListEntries(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to list entries", err)
		return
	}
	err = s.templates.TmplSearch.Execute(w, entries)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

// show login Form
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	_, span = tracer.Start(ctx, "server.loginHandler")
	defer span.End()

	err := s.templates.TmplLogin.Execute(w, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

// show signup Form
func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	_, span = tracer.Start(ctx, "server.signupHandler")
	defer span.End()

	err := s.templates.TmplSignUp.Execute(w, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.dashboardHandler")
	defer span.End()

	session, err := r.Cookie("session")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "could not find cookie", err)
		return
	}
	tokenValue, err := s.tokenstore.GetTokenValue(ctx, session)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to get token value", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, tokenValue)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to get user", err)
		return
	}
	user.Entry, err = s.bookstore.GetEntryByID(ctx, tokenValue)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to get entry", err)
		return
	}

	err = s.templates.TmplDashboard.Execute(w, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) createHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	_, span = tracer.Start(ctx, "server.createHandler")
	defer span.End()

	err := s.templates.TmplCreate.Execute(w, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) verifyHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	_, span = tracer.Start(ctx, "server.verifyHandler")
	defer span.End()

	err := s.templates.TmplVerification.Execute(w, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.adminHandler")
	defer span.End()

	users, err := s.userstore.ListUser(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		s.log.ErrorContext(ctx, "failed to list user", err)
		return
	}
	err = s.templates.TmplAdmin.Execute(w, &users)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) forgotHandler(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	_, span = tracer.Start(ctx, "server.forgotHandler")
	defer span.End()

	err := s.templates.TmplForgot.Execute(w, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}

func (s *Server) createEntry(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.createEntry")
	defer span.End()

	err := r.ParseForm()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse form", err)
		return
	}
	session, err := r.Cookie("session")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to find cookie", err)
		return
	}
	userID, err := s.tokenstore.GetTokenValue(ctx, session)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get token value", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", err)
		return
	}
	newEntry := model.GuestbookEntry{Name: user.Name, Message: html.EscapeString(r.FormValue("message")), UserID: user.ID}

	_, err = s.bookstore.CreateEntry(ctx, &newEntry)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to create entry", err)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

func (s *Server) changeUserData(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.changeUserData")
	defer span.End()

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", err)
		return
	}
	err = s.templates.TmplDashboardUser.ExecuteTemplate(w, "user-update", user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", err)
		return
	}
}
