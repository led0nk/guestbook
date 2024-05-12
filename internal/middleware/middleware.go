package middleware

import (
	"log/slog"
	"net/http"

	db "github.com/led0nk/guestbook/internal/database"
	sloghttp "github.com/samber/slog-http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("github.com/led0nk/guestbook/internal/middleware")

func Auth(t db.TokenStore, logger *slog.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var span trace.Span
			ctx := r.Context()
			ctx, span = tracer.Start(ctx, "middleware.Auth")
			defer span.End()

			session, err := r.Cookie("session")
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not find cookie", "error", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(ctx, session.Value)
			if !isValid {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not validate token", "error", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			cookie, err := t.Refresh(ctx, session.Value)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not refresh token", "error", err)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)
			logger.Info("authentication middleware", "status", "done")
			h.ServeHTTP(w, r)
		})
	}
}

func AdminAuth(t db.TokenStore, u db.UserStore, logger *slog.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var span trace.Span
			ctx := r.Context()
			ctx, span = tracer.Start(ctx, "middleware.AdminAuth")
			defer span.End()

			session, err := r.Cookie("session")
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not find cookie", "error", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(ctx, session.Value)
			if !isValid {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not validate token", "error", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			cookie, err := t.Refresh(ctx, session.Value)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "could not refresh token", "error", err)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)

			logger.Info("admin middleware", "status", "done")

			h.ServeHTTP(w, r)
		})
	}
}

func SlogAddTraceAttributes() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sloghttp.AddCustomAttributes(r,
				slog.String("trace-id", trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()),
			)
			sloghttp.AddCustomAttributes(r,
				slog.String("span-id", trace.SpanFromContext(r.Context()).SpanContext().SpanID().String()),
			)
			h.ServeHTTP(w, r)
		})
	}
}
