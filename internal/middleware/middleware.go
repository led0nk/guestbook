package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/rs/zerolog"
)

type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (rec *ResponseRecorder) WriteHeader(statusCode int) {
	rec.StatusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

// logging middleware
func Logger(logger zerolog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &ResponseRecorder{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
			}

			logger.Info().Str(r.Method, r.URL.String()).Msg(strconv.Itoa(rec.StatusCode))
			next.ServeHTTP(rec, r)
		})
	}
}

// authentication middleware, check for session values -> redirect
func Auth(t db.TokenStore, logger zerolog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil {
				logger.Err(errors.New("cookie")).Msg(err.Error())
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(session.Value)
			if !isValid {
				logger.Err(errors.New("token")).Msg(err.Error())
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			cookie, err := t.Refresh(session.Value)
			if err != nil {
				logger.Err(errors.New("token")).Msg(err.Error())
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)
			logger.Info().Str("auth-mw", "done").Msg("")
			next.ServeHTTP(w, r)
		})
	}
}

func AdminAuth(t db.TokenStore, u db.UserStore, logger zerolog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil {
				logger.Err(errors.New("cookie")).Msg(err.Error())
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(session.Value)
			if !isValid {
				logger.Err(errors.New("token")).Msg(err.Error())
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			cookie, err := t.Refresh(session.Value)
			if err != nil {
				logger.Err(errors.New("token")).Msg(err.Error())
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)

			logger.Info().Str("admin-mw", "done").Msg("")

			next.ServeHTTP(w, r)
		})
	}
}
