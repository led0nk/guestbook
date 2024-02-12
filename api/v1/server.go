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
	authMiddleware.HandleFunc("/verify", s.verifyHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/verify", s.verifyAuth()).Methods(http.MethodPost)
	// routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", s.dashboardHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search", s.searchHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", s.createEntry()).Methods(http.MethodPost)
	// routing through admincheck via /admin
	adminMiddleware.HandleFunc("/dashboard", s.adminHandler).Methods(http.MethodGet)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.deleteUser()).Methods(http.MethodDelete)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.updateUser()).Methods(http.MethodPost)
	adminMiddleware.HandleFunc("/dashboard/{ID}", s.saveUser()).Methods(http.MethodPut)
	adminMiddleware.HandleFunc("/dashboard/{ID}/verify", s.resendVer()).Methods(http.MethodPut)
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

// TODO: implement r.url q= and list entries after Post method (new Handler)
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplSearch.Execute(w, nil)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
		return
	}
}

// show login Form
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplLogin.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
		return
	}
}

// show signup Form
func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplSignUp.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
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
			s.log.Warn("Template Error: ", err)
			return
		}

	}
}

func (s *Server) createHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplCreate.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
		return
	}
}

func (s *Server) verifyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.templates.TmplVerification.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
		return
	}
}

func (s *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
	users, _ := s.userstore.ListUser()
	err := s.templates.TmplAdmin.Execute(w, &users)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		s.log.Warn("Template Error: ", err)
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
			s.log.Warn("Entry Error: ", err)
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
			s.log.Warn("User Error: ", error)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			s.log.Warn("Hashing Error: ", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		cookie, err := s.tokenstore.CreateToken("session", user.ID, utils.FormValueBool(r.FormValue("Rememberme")))
		if err != nil {
			s.log.Warn("Token Error: ", err)
			return
		}

		http.SetCookie(w, cookie)
		if user.IsAdmin {
			http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		}
		http.Redirect(w, r, "/user/verify", http.StatusFound)
	}
}

// logoutAuth and deleting session-cookie
func (s *Server) logoutAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				s.log.Warn(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
		}
		userID, _ := s.tokenstore.GetTokenValue(cookie)
		err = s.tokenstore.DeleteToken(userID)
		if err != nil {
			s.log.Warn("Token Error: ", err)
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
			s.log.Warn("Input Error: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}
		joinedName := strings.Join([]string{utils.Capitalize(r.FormValue("firstname")), utils.Capitalize(r.FormValue("lastname"))}, " ")
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
		_, usererr := s.userstore.CreateUser(&newUser)
		if usererr != nil {
			s.log.Warn("User error: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			w.WriteHeader(http.StatusUnauthorized)
		}

		err = s.mailer.SendVerMail(&newUser, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) verifyAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		session, _ := r.Cookie("session")
		s.log.Debug(session)
		s.log.Debug(session.Value)
		userID, err := s.tokenstore.GetTokenValue(session)
		if err != nil {
			s.log.Warn("Token Error: ", err)
			return
		}
		ok, err := s.userstore.CodeValidation(userID, r.FormValue("code"))
		if !ok {
			http.Redirect(w, r, "/user/verify", http.StatusFound)
			s.log.Info("User Error: ", err)
			return
		}
		if err != nil {
			s.log.Error("User Error:", err)
			return
		}
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

func (s *Server) deleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		err = s.userstore.DeleteUser(ID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		// http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	}
}

// TODO: User Template with input Form for editing
func (s *Server) updateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error", err)
			return
		}
		err = s.templates.TmplAdminInput.Execute(w, &user)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

// save updated User data
func (s *Server) saveUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}

		updatedUser := model.User{
			ID:               user.ID,
			Email:            r.FormValue("Email"),
			Name:             user.Name,
			Password:         user.Password,
			IsAdmin:          utils.FormValueBool(r.FormValue("Admin")),
			IsVerified:       utils.FormValueBool(r.FormValue("Verified")),
			VerificationCode: user.VerificationCode,
			ExpirationTime:   user.ExpirationTime,
		}
		err = s.userstore.UpdateUser(&updatedUser)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		err = s.templates.TmplAdminUser.Execute(w, &updatedUser)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

func (s *Server) resendVer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		user.VerificationCode = utils.RandomString(6)
		user.ExpirationTime = time.Now().Add(time.Minute * 5)
		err = s.mailer.SendVerMail(user, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		err = s.userstore.UpdateUser(user)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
	}
}
