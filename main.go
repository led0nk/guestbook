package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/led0nk/guestbook/db"
	"github.com/led0nk/guestbook/db/jsondb"
	"github.com/led0nk/guestbook/model"
	"github.com/led0nk/guestbook/token"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

var cookies = sessions.NewCookieStore([]byte("secret"))

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})

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
	var gueststore db.GuestBookStorage
	var userstore db.UserStorage
	var tokenservice token.TokenService
	switch u.Scheme {
	case "file":
		log.Info("opening:", u.Hostname())
		bookStorage, _ := jsondb.CreateBookStorage("./entries.json")
		userStorage, _ := jsondb.CreateUserStorage("./user.json")
		tokenStorage, _ := token.CreateTokenService()
		gueststore = bookStorage
		userstore = userStorage
		tokenservice = tokenStorage
	default:
		panic("bad storage")
	}
	//logMiddleware := mux.NewRouter()
	authMiddleware := mux.NewRouter().PathPrefix("/user").Subrouter()
	authMiddleware.Use(authHandler)
	router.Use(logHandler)
	router.PathPrefix("/user").Handler(authMiddleware)
	//routing
	router.HandleFunc("/", handlePage(gueststore)).Methods(http.MethodGet)
	router.HandleFunc("/submit", submit(gueststore)).Methods(http.MethodPost)
	router.HandleFunc("/", delete(gueststore)).Methods(http.MethodPost)
	router.HandleFunc("/login", loginHandler()).Methods(http.MethodGet)
	router.HandleFunc("/login", loginAuth(tokenservice, userstore)).Methods(http.MethodPost)
	router.HandleFunc("/search", searchHandler()).Methods(http.MethodGet)
	router.HandleFunc("/logout", logout(userstore)).Methods(http.MethodGet)
	router.HandleFunc("/signup", signupHandler()).Methods(http.MethodGet)
	router.HandleFunc("/signupauth", signupAuth(userstore)).Methods(http.MethodPost)
	//routing through authentication via /user
	authMiddleware.HandleFunc("/dashboard", dashboardHandler(tokenservice, userstore, gueststore)).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", createHandler()).Methods(http.MethodGet)
	authMiddleware.HandleFunc("/create", createEntry(userstore, gueststore)).Methods(http.MethodPost)
	log.Info("listening to: ")
	serverr := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(serverr)
	}
}

// logging middleware
func logHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var arrow string
		switch r.Method {
		case http.MethodPost:
			post := " <----- "
			arrow = post
		default:
			others := " -----> "
			arrow = others
		}
		log.Info(r.URL, arrow, "["+r.Method+"]")
		next.ServeHTTP(w, r)
	})
}

// authentication middleware, check for session values -> redirect
func authHandler(t token.TokenService, next http.Handler) http.Handler {
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
		isValid, exp, err := t.Valid(session.Value, session.Expires)
		if !isValid {
			log.Error("invalid Token", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session.Expires = exp
		http.SetCookie(w, session)

		log.Info("authentication successfull")
		next.ServeHTTP(w, r)
	})
}

// hands over Entries to Handler and prints them out in template
func handlePage(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/content.html")
		searchName := r.URL.Query().Get("q")
		var entries []*model.GuestbookEntry
		if searchName != "" {
			entries, _ = s.GetEntryByName(searchName)
		} else {
			entries, _ = s.ListEntries()
		}
		err := tmplt.Execute(w, &entries)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	}
}

// submits guestbook entry (name, message)
func submit(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newEntry := model.GuestbookEntry{Name: r.FormValue("name"), Message: r.FormValue("message")}
		if newEntry.Name == "" {
			return
		}
		s.CreateEntry(&newEntry)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func delete(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.ParseForm()
		strUuid := r.Form.Get("Delete")
		uuidStr, _ := uuid.Parse(strUuid)

		deleteEntry := model.GuestbookEntry{ID: uuidStr}
		s.DeleteEntry(deleteEntry.ID)
		http.Redirect(w, r, "/", http.StatusFound)

	}
}

func searchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/search.html")

		err := tmplt.Execute(w, nil)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
	}
}

// show login Form
func loginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/login.html")
		err := tmplt.Execute(w, nil)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
	}
}

// show signup Form
func signupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/signup.html")
		err := tmplt.Execute(w, nil)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
	}
}

// login authentication and check if user exists
func loginAuth(t token.TokenService, u db.UserStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		user, error := u.GetUserByEmail(email)
		if error != nil {
			fmt.Println("cannot access right hashpassword", error)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tokenString, tokenExpire, err := t.CreateToken(user.ID)
		log.Info(tokenExpire)
		if err != nil {
			log.Error(err)
			return
		}
		cookie := http.Cookie{
			Name:    "session",
			Value:   tokenString,
			Path:    "/",
			Expires: tokenExpire,
			//MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

// logout and deleting session-cookie
func logout(u db.UserStorage) http.HandlerFunc {
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
			return
		}
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func dashboardHandler(t token.TokenService, u db.UserStorage, b db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/loggedinheader.html", "templates/dashboard.html")

		session, _ := r.Cookie("session")
		tokenValue, _ := t.GetTokenValue(session)
		user, _ := u.GetUserByID(tokenValue)
		log.Info(tokenValue)

		tmplt.Execute(w, user)
	}
}

func testHandler(t token.TokenService, u db.UserStorage, b db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/loggedinheader.html", "templates/dashboard.html")

		session, _ := r.Cookie("session")
		tokenValue, _ := t.GetTokenValue(session)

		user, _ := u.GetUserByID(tokenValue)
		log.Info(tokenValue)

		tmplt.Execute(w, user)
	}
}

// signup authentication and validation of user input
func signupAuth(u db.UserStorage) http.HandlerFunc {
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
		u.CreateUser(&newUser)
	}
}

func createHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/loggedinheader.html", "templates/create.html")
		tmplt.Execute(w, nil)
	}
}

func createEntry(u db.UserStorage, b db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/loggedinheader.html", "templates/create.html")
		r.ParseForm()
		session, _ := cookies.Get(r, "session")
		sessionUserID := session.Values["ID"].(string)
		userID, _ := uuid.Parse(sessionUserID)
		user, _ := u.GetUserByID(userID)

		newEntry := model.GuestbookEntry{Name: user.Name, Message: r.FormValue("message"), UserID: user.ID}
		b.CreateEntry(&newEntry)
		tmplt.Execute(w, nil)
	}
}
