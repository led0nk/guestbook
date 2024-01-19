package middleware

import (
	"net/http"

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
func Logger(next http.Handler) http.Handler {
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

// authentication middleware, check for session values -> redirect
func (s *Store) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("session")
		if err != nil {
			log.Error("there are no cookies of type session")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		exists := session.Value
		if exists == "" {
			log.Info("authentication failed, no tokens available for session")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		isValid, err := s.tokenstore.Valid(session.Value)
		if !isValid {
			log.Error("Tokenerror: ", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		cookie, err := s.tokenstore.Refresh(session.Value)
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
