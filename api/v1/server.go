package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	log        logrus.FieldLogger
	mw         []mux.MiddlewareFunc
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewServer(
	address string,
	logger logrus.FieldLogger,
	bStore db.GuestBookStore,
	uStore db.UserStore,
	tStore db.TokenStore,
	middleware ...mux.MiddlewareFunc,
) *Server {
	return &Server{
		addr:       address,
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
	authMiddleware.Use(middleware.Auth(s.tokenstore))
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
	// routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", s.dashboardHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", createHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search", s.searchHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createEntry()).Methods(http.MethodPost)
	// log.Info("listening to: ")

	srv := &http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	err := srv.ListenAndServe()
	if err != nil {
		// log.Fatal(err)
	}
}

// hands over Entries to Handler and prints them out in template
func (s *Server) handlePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		searchName := r.URL.Query().Get("q")
		var entries []*model.GuestbookEntry
		if searchName != "" {
			entries, _ = s.bookstore.GetEntryByName(searchName)
		} else {
			entries, _ = s.bookstore.ListEntries()
		}

		tmp := templates.NewTemplateHandler()
		err := tmp.TmplHome.Execute(w, &entries)
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
		s.bookstore.DeleteEntry(deleteEntry.ID)
		http.Redirect(w, r, "/", http.StatusFound)

	}
}

// TODO: implement r.url q= and list entries after Post method (new Handler)
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplSearch.Execute(w, nil)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show login Form
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplLogin.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show signup Form
func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplSignUp.Execute(w, nil)
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

		tmp := templates.NewTemplateHandler()
		err := tmp.TmplDashboard.Execute(w, user)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	}
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplCreate.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// login authentication and check if user exists
func (s *Server) loginAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		user, error := s.userstore.GetUserByEmail(email)
		if error != nil {
			s.log.Error("cannot access right hashpassword", error)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			s.log.Error("error while comparing password", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		cookie, err := s.tokenstore.CreateToken(user.ID)
		if err != nil {
			s.log.Error(err)
			return
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
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
		s.tokenstore.DeleteToken(userID)
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
		newUser := model.User{Email: r.FormValue("email"), Name: joinedName, Password: hashedpassword, IsAdmin: false}
		debug, usererr := s.userstore.CreateUser(&newUser)
		if usererr != nil {
			s.log.Error("creation error: ", err)
			http.Redirect(w, r, "signup", http.StatusFound)
			w.WriteHeader(http.StatusUnauthorized)
		}
		debuguser, _ := s.userstore.GetUserByID(debug)
		s.log.Debug(debuguser)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) createEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		session, _ := r.Cookie("session")
		userID, _ := s.tokenstore.GetTokenValue(session)
		user, _ := s.userstore.GetUserByID(userID)

		newEntry := model.GuestbookEntry{Name: user.Name, Message: r.FormValue("message"), UserID: user.ID}

		s.bookstore.CreateEntry(&newEntry)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}
