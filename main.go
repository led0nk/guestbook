package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	templates "github.com/led0nk/guestbook/internal"
	db "github.com/led0nk/guestbook/internal/database"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/model"
	"github.com/led0nk/guestbook/token"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

type Store struct {
	bookstore  db.GuestBookStore
	userstore  db.UserStore
	tokenstore db.TokenStore
}

func NewHandler(bookstore db.GuestBookStore, userstore db.UserStore, tokenstore db.TokenStore) *Store {
	return &Store{
		bookstore:  bookstore,
		userstore:  userstore,
		tokenstore: tokenstore,
	}
}

func main() {

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)
	router := mux.NewRouter()

	var (
		//addr     = flag.String("addr", "localhost:8080", "server port")
		entryStr = flag.String("entrydata", "file://entries.json", "link to entry-database")
		//userStr  = flag.String("userdata", "file://user.json", "link to user-database")
	)
	flag.Parse()
	u, err := url.Parse(*entryStr)
	if err != nil {
		panic(err)
	}
	log.Info(u)
	var guestbookStore db.GuestBookStore
	var userStore db.UserStore
	var tokenStore db.TokenStore
	switch u.Scheme {
	case "file":
		log.Info("opening:", u.Hostname())
		bookStorage, _ := jsondb.CreateBookStorage("./entries.json")
		userStorage, _ := jsondb.CreateUserStorage("./user.json")
		tokenStorage, _ := token.CreateTokenService()
		guestbookStore = bookStorage
		userStore = userStorage
		tokenStore = tokenStorage

	default:
		panic("bad storage")
	}

	storeHandler := NewHandler(guestbookStore, userStore, tokenStore)

	//logMiddleware := mux.NewRouter()
	authMiddleware := mux.NewRouter().PathPrefix("/user").Subrouter()
	authMiddleware.Use(storeHandler.authHandler)
	router.Use(logHandler)
	router.PathPrefix("/user").Handler(authMiddleware)
	//routing
	router.HandleFunc("/", storeHandler.handlePage()).Methods(http.MethodGet)
	router.HandleFunc("/submit", storeHandler.submit).Methods(http.MethodPost)
	router.HandleFunc("/", storeHandler.delete()).Methods(http.MethodPost)
	router.HandleFunc("/login", storeHandler.loginHandler).Methods(http.MethodGet)
	router.HandleFunc("/login", storeHandler.loginAuth()).Methods(http.MethodPost)
	router.HandleFunc("/logout", storeHandler.logout()).Methods(http.MethodGet)
	router.HandleFunc("/signup", storeHandler.signupHandler).Methods(http.MethodGet)
	router.HandleFunc("/signupauth", storeHandler.signupAuth()).Methods(http.MethodPost)
	//routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", storeHandler.dashboardHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", createHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/search", storeHandler.searchHandler).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", storeHandler.createEntry()).Methods(http.MethodPost)
	log.Info("listening to: ")
	serverr := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(serverr)
	}
}

type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (rec *ResponseRecorder) WriteHeader(statusCode int) {
	rec.StatusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

// logging middleware
func logHandler(next http.Handler) http.Handler {
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
func (s *Store) authHandler(next http.Handler) http.Handler {
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

// hands over Entries to Handler and prints them out in template
func (s *Store) handlePage() http.HandlerFunc {
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

// submits guestbook entry (name, message)
func (s *Store) submit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	newEntry := model.GuestbookEntry{Name: r.FormValue("name"), Message: r.FormValue("message")}
	if newEntry.Name == "" {
		return
	}
	s.bookstore.CreateEntry(&newEntry)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Store) delete() http.HandlerFunc {
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
func (s *Store) searchHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplSearch.Execute(w, nil)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show login Form
func (s *Store) loginHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplLogin.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// show signup Form
func (s *Store) signupHandler(w http.ResponseWriter, r *http.Request) {
	tmp := templates.NewTemplateHandler()
	err := tmp.TmplSignUp.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
}

// login authentication and check if user exists
func (s *Store) loginAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		user, error := s.userstore.GetUserByEmail(email)
		if error != nil {
			fmt.Println("cannot access right hashpassword", error)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		cookie, err := s.tokenstore.CreateToken(user.ID)
		if err != nil {
			log.Error(err)
			return
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

// logout and deleting session-cookie
func (s *Store) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				log.Println(err)
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

func (s *Store) dashboardHandler() http.HandlerFunc {
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

// signup authentication and validation of user input
func (s *Store) signupAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := jsondb.ValidateUserInput(r.Form)
		if err != nil {
			fmt.Println("user form not valid:", err)
			http.Redirect(w, r, "/signup", http.StatusBadRequest)
			return
		}
		joinedName := strings.Join([]string{r.FormValue("firstname"), r.FormValue("lastname")}, " ")
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), 14)
		newUser := model.User{Email: r.FormValue("email"), Name: joinedName, Password: hashedpassword}
		s.userstore.CreateUser(&newUser)
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

func (s *Store) createEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		session, _ := r.Cookie("session")
		userID, _ := s.tokenstore.GetTokenValue(session)
		user, _ := s.userstore.GetUserByID(userID)

		newEntry := model.GuestbookEntry{Name: user.Name, Message: r.FormValue("message"), UserID: user.ID}
		s.bookstore.CreateEntry(&newEntry)
		tmp := templates.NewTemplateHandler()
		err := tmp.TmplCreate.Execute(w, user)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	}
}
