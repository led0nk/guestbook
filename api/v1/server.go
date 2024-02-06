package v1

import (
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/led0nk/guestbook/cmd/utils"
	templates "github.com/led0nk/guestbook/internal"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/middleware"
	"github.com/led0nk/guestbook/internal/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	addr       string
	mailer     Mailerservice
	templates  *templates.TemplateHandler
	log        logrus.FieldLogger
	mw         []mux.MiddlewareFunc
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewServer(
	address string,
	mailer Mailerservice,
	templates *templates.TemplateHandler,
	logger logrus.FieldLogger,
	bStore db.GuestBookStore,
	uStore db.UserStore,
	tStore db.TokenStore,
	middleware ...mux.MiddlewareFunc,
) *Server {
	return &Server{
		addr:       address,
		mailer:     mailer,
		templates:  templates,
		log:        logger,
		mw:         middleware,
		bookstore:  bStore,
		userstore:  uStore,
		tokenstore: tStore,
	}
}

func (s *Server) ServeHTTP() {
	// has to be called in main including above initialisations
	router := mux.NewRouter()

	authMiddleware := mux.NewRouter().PathPrefix("/user").Subrouter()
	adminMiddleware := mux.NewRouter().PathPrefix("/admin").Subrouter()
	authMiddleware.Use(middleware.Auth(s.tokenstore))
	adminMiddleware.Use(middleware.AdminAuth(s.tokenstore, s.userstore))
	router.Use(middleware.Logger())
	router.PathPrefix("/user").Handler(authMiddleware)
	// routing
	router.HandleFunc("/", s.handlePage()).Methods(http.MethodGet)
	router.HandleFunc("/", s.delete()).Methods(http.MethodPost)
	router.HandleFunc("/login", s.loginHandler).Methods(http.MethodGet)
	router.HandleFunc("/login", s.loginAuth()).Methods(http.MethodPost)
	router.HandleFunc("/logout", s.logout()).Methods(http.MethodGet)
	router.HandleFunc("/signup", s.signupHandler).Methods(http.MethodGet)
	router.HandleFunc("/signup", s.signupAuth()).Methods(http.MethodPost)
	router.HandleFunc("/verify", s.verifyHandler).Methods(http.MethodGet)
	router.HandleFunc("/verify", s.verifyAuth()).Methods(http.MethodPost)
	// routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", s.dashboardHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search", s.searchHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createEntry()).Methods(http.MethodPost)
	// routing through admincheck via /admin
	adminMiddleware.HandleFunc("/admin", s.adminHandler).Methods(http.MethodGet)
	// log.Info("listening to: ")

	srv := &http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	err := srv.ListenAndServe()
	if err != nil {
		s.log.Fatal(err)
	}
}

// hands over Entries to Handler and prints them out in template
func (s *Server) handlePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//TODO for Search
		searchName := r.URL.Query().Get("q")
		var entries []*model.GuestbookEntry
		if searchName != "" {
			entries, _ = s.bookstore.GetEntryByName(searchName)
		} else {
			entries, _ = s.bookstore.ListEntries()
		}

		err := s.templates.TmplHome.Execute(w, &entries)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	}
}

func (s *Server) delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.ParseForm()
		strUuid := r.Form.Get("Delete")
		uuidStr, _ := uuid.Parse(strUuid)

		deleteEntry := model.GuestbookEntry{ID: uuidStr}
		err := s.bookstore.DeleteEntry(deleteEntry.ID)
		if err != nil {
			s.log.Error("Entry Error: ", err)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)

	}
}

// TODO: implement r.url q= and list entries after Post method (new Handler)
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplSearch.Execute(w, nil)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show login Form
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplLogin.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show signup Form
func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplSignUp.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

func (s *Server) dashboardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := r.Cookie("session")
		tokenValue, _ := s.tokenstore.GetTokenValue(session)
		user, _ := s.userstore.GetUserByID(tokenValue)
		user.Entry, _ = s.bookstore.GetEntryByID(tokenValue)

		err := s.templates.TmplDashboard.Execute(w, user)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	}
}

func (s *Server) createHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplCreate.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

func (s *Server) verifyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplVerification.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	s.log.Info(mux.Vars(r))
}

func (s *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
    return func(w http.ResponseWriter, r *http.Request){
    err := s.templates.
  }
}

func (s *Server) createEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		session, _ := r.Cookie("session")
		userID, _ := s.tokenstore.GetTokenValue(session)
		user, _ := s.userstore.GetUserByID(userID)
		newEntry := model.GuestbookEntry{Name: user.Name, Message: html.EscapeString(r.FormValue("message")), UserID: user.ID}

		_, err := s.bookstore.CreateEntry(&newEntry)
		if err != nil {
			s.log.Error("Entry Error: ", err)
			return
		}
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

// login authentication and check if user exists
func (s *Server) loginAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		user, error := s.userstore.GetUserByEmail(email)
		if error != nil {
			s.log.Error("User Error: ", error)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			s.log.Error("Hashing Error: ", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		cookie, err := s.tokenstore.CreateToken("session", user.ID)
		if err != nil {
			s.log.Error(err)
			return
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/verify", http.StatusFound)
	}
}

// logout and deleting session-cookie
func (s *Server) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				s.log.Error(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
		}
		userID, _ := s.tokenstore.GetTokenValue(cookie)
		err = s.tokenstore.DeleteToken(userID)
		if err != nil {
			s.log.Error("Token Error: ", err)
			return
		}
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// signup authentication and validation of user input
func (s *Server) signupAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := jsondb.ValidateUserInput(r.Form)
		if err != nil {
			s.log.Error("user form not valid: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}
		joinedName := strings.Join([]string{r.FormValue("firstname"), r.FormValue("lastname")}, " ")
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), 14)
		newUser := model.User{
			Email:            html.EscapeString(r.FormValue("email")),
			Name:             html.EscapeString(joinedName),
			Password:         hashedpassword,
			IsAdmin:          false,
			IsVerified:       false,
			VerificationCode: utils.RandomString(6),
			ExpirationTime:   time.Now().Add(time.Minute * 5),
		}
		userID, usererr := s.userstore.CreateUser(&newUser)
		if usererr != nil {
			s.log.Error("creation error: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			w.WriteHeader(http.StatusUnauthorized)
		}

		var cookie *http.Cookie
		cookie, err = s.tokenstore.CreateToken("verification", userID)
		if err != nil {
			s.log.Error("Token Error: ", err)
			return
		}
		err = s.mailer.SendVerMail(&newUser, s.templates)
		if err != nil {
			s.log.Error("Mailer Error: ", err)
			return
		}
		http.SetCookie(w, cookie)
		//joinedPath, _ := url.JoinPath("/verify", userID.String())
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) verifyAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		s.log.Info("test")
		session, _ := r.Cookie("session")
		userID, _ := s.tokenstore.GetTokenValue(session)
		ok, err := s.userstore.CodeValidation(userID, r.FormValue("code"))
		if !ok {
			http.Redirect(w, r, "/verify", http.StatusFound)
			s.log.Info("User: Validation Code not correct")
			return
		}
		if err != nil {
			s.log.Error("User Error:", err)
			return
		}
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}
