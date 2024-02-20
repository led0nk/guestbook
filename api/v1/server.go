package v1

import (
	"errors"
	"html"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	templates "github.com/led0nk/guestbook/internal"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/middleware"
	"github.com/led0nk/guestbook/internal/model"
	"github.com/rs/zerolog"
)

type Server struct {
	addr       string
	mailer     Mailerservice
	templates  *templates.TemplateHandler
	log        zerolog.Logger
	mw         []mux.MiddlewareFunc
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewServer(
	address string,
	mailer Mailerservice,
	templates *templates.TemplateHandler,
	logger zerolog.Logger,
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
	authMiddleware.Use(middleware.Auth(s.tokenstore, s.log))
	adminMiddleware.Use(middleware.AdminAuth(s.tokenstore, s.userstore, s.log))
	router.Use(s.mw[0])
	router.PathPrefix("/user").Handler(authMiddleware)
	router.PathPrefix("/admin").Handler(adminMiddleware)
	// routing
	router.HandleFunc("/", s.handlePage()).Methods(http.MethodGet)
	// router.HandleFunc("/", s.delete()).Methods(http.MethodPost)
	router.HandleFunc("/login", s.loginHandler).Methods(http.MethodGet)
	router.HandleFunc("/login", s.loginAuth()).Methods(http.MethodPost)
	router.HandleFunc("/logout", s.logoutAuth()).Methods(http.MethodGet)
	router.HandleFunc("/signup", s.signupHandler).Methods(http.MethodGet)
	router.HandleFunc("/signup", s.signupAuth()).Methods(http.MethodPost)
	router.HandleFunc("/forgot-pw", s.forgotHandler).Methods(http.MethodGet)
	router.HandleFunc("/forgot-pw", s.forgotPW()).Methods(http.MethodPost)
	authMiddleware.HandleFunc("/verify", s.verifyHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/verify", s.verifyAuth()).Methods(http.MethodPost)
	// routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", s.dashboardHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/dashboard/{ID}", s.changeUserData()).Methods(http.MethodPost)
	authMiddleware.HandleFunc("/dashboard/{ID}", s.submitUserData()).Methods(http.MethodPut)
	authMiddleware.HandleFunc("/create", s.createHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search", s.searchHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search/", s.search()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createEntry()).Methods(http.MethodPost)
	authMiddleware.HandleFunc("/dashboard/{ID}/password-reset", s.passwordReset()).Methods(http.MethodPut)
	// routing through admincheck via /admin
	adminMiddleware.HandleFunc("/dashboard", s.adminHandler).Methods(http.MethodGet)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.deleteUser()).Methods(http.MethodDelete)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.updateUser()).Methods(http.MethodPost)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.saveUser()).Methods(http.MethodPut)
	adminMiddleware.HandleFunc("/dashboard/{ID}/verify", s.resendVer()).Methods(http.MethodPut)
	adminMiddleware.HandleFunc("/dashboard/{ID}/password-reset", s.passwordReset()).Methods(http.MethodPut)
	// log.Info("listening to: ")

	srv := &http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	err := srv.ListenAndServe()
	if err != nil {
		s.log.Fatal().Err(err).Msg("")
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
			s.log.Error().Err(errors.New("template")).Msg("")
			return
		}

	}
}

// TODO: implement r.url q= and list entries after Post method (new Handler)
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := s.bookstore.ListEntries()
	if err != nil {
		s.log.Err(errors.New("user")).Msg(err.Error())
		return
	}
	err = s.templates.TmplSearch.Execute(w, entries)
	if err != nil {
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
	}
}

// show login Form
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplLogin.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
	}
}

// show signup Form
func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplSignUp.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
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
			s.log.Err(errors.New("template")).Msg(err.Error())
			return
		}

	}
}

func (s *Server) createHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplCreate.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
	}
}

func (s *Server) verifyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplVerification.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
	}
}

func (s *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
	users, _ := s.userstore.ListUser()
	err := s.templates.TmplAdmin.Execute(w, &users)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
	}
}

func (s *Server) forgotHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplForgot.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Err(errors.New("template")).Msg(err.Error())
		return
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
			s.log.Err(errors.New("entry")).Msg(err.Error())
			return
		}
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

func (s *Server) changeUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Err(errors.New("uuid")).Msg(err.Error())
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Err(errors.New("user")).Msg(err.Error())
			return
		}
		err = s.templates.TmplDashboardUser.ExecuteTemplate(w, "user-update", user)
		if err != nil {
			s.log.Err(errors.New("template")).Msg(err.Error())
			return
		}
	}
}
