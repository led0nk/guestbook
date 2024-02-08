package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
	db "github.com/led0nk/guestbook/internal/database"
	log "github.com/sirupsen/logrus"
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
func Logger() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var arrow string
			rec := &ResponseRecorder{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
			}

			switch r.Method {
			case http.MethodPost:
				post := " <----- "
				arrow = post
			default:
				others := " -----> "
				arrow = others
			}
			log.Info("[", rec.StatusCode, "]", r.URL, arrow, "["+r.Method+"]") //StatusCode in progress, not working yet
			next.ServeHTTP(rec, r)
		})
	}
}

// authentication middleware, check for session values -> redirect
func Auth(t db.TokenStore) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil {
				log.Error("there are no cookies of type session")
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(session.Value)
			if !isValid {
				log.Error("Tokenerror: ", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			cookie, err := t.Refresh(session.Value)
			if err != nil {
				log.Error("Error Refreshing: ", err)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)
			log.Info("authMiddleware done")
			next.ServeHTTP(w, r)
		})
	}
}

func AdminAuth(t db.TokenStore, u db.UserStore) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil {
				log.Error("there are no cookies of type session")
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			isValid, err := t.Valid(session.Value)
			if !isValid {
				log.Warn("Token Error: ", err)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			log.Debug(session.Value)

			userID, err := t.GetTokenValue(session)
			if err != nil {
				log.Warn("Token Error: ", err)
				return
			}
			log.Debug(userID)
			user, err := u.GetUserByID(userID)
			if err != nil {
				log.Warn("User Error: ", err)
				return
			}
			if !user.IsAdmin {
				log.Warn("User Error: User is not a registered admin!")
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			cookie, err := t.Refresh(session.Value)
			if err != nil {
				log.Warn("Error Refreshing: ", err)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			http.SetCookie(w, cookie)
			log.Info("Admin checked")
			next.ServeHTTP(w, r)
		})
	}
}
