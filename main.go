package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/led0nk/guestbook/db"
	"github.com/led0nk/guestbook/db/jsondb"
	"github.com/led0nk/guestbook/model"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

// var filename string = "./jsondb/entries.json"
var tmplt *template.Template

func main() {
	m := http.NewServeMux()

	var (
		addr     = flag.String("addr", ":8080", "server port")
		entryStr = flag.String("entrydata", "file://entries.json", "link to entry-database")
		//userStr  = flag.String("userdata", "file://user.json", "link to user-database")
	)
	flag.Parse()

	u, err := url.Parse(*entryStr)
	if err != nil {
		panic(err)
	}
	log.Print(u)
	var gueststore db.GuestBookStorage
	var userstore db.UserStorage
	switch u.Scheme {
	case "file":
		log.Println("opening:", u.Hostname())
		bookStorage, _ := jsondb.CreateBookStorage("./entries.json")
		userStorage, _ := jsondb.CreateUserStorage("./user.json")
		gueststore = bookStorage
		userstore = userStorage
	default:
		panic("bad storage")
	}

	//placeholder
	m.HandleFunc("/", handlePage(gueststore))
	m.HandleFunc("/submit", submit(gueststore))
	m.HandleFunc("/delete", delete(gueststore))
	m.HandleFunc("/login", login())
	m.HandleFunc("/loginauth", loginAuth(userstore))
	m.HandleFunc("/signup", signupHandler())
	m.HandleFunc("/signupauth", signupAuth(userstore))

	srv := http.Server{
		Handler:      m,
		Addr:         *addr,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	fmt.Println("server started on port", addr)
	fmt.Println(time.Now())
	error := srv.ListenAndServe()
	log.Fatal(error)
}

// hands over Entries to Handler and prints them out in template
func handlePage(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		tmplt, _ = template.ParseFiles("index.html", "header.html", "content.html")
		searchName := r.URL.Query().Get("q")
		var entries []*model.GuestbookEntry
		if searchName != "" {
			entries, _ = s.GetEntryByName(searchName)
		} else {
			entries, _ = s.ListEntries()
		}
		err := tmplt.Execute(w, &entries)
		if err != nil {
			fmt.Println("error when executing template", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// submits guestbook entry (name, message)
func submit(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("method:", r.Method)
		if r.Method == "GET" {
			tmplt, _ := template.ParseFiles("index.html", "header.html", "content.html")
			tmplt.Execute(w, nil)
		} else {
			r.ParseForm()
			newEntry := model.GuestbookEntry{Name: r.FormValue("name"), Message: r.FormValue("message")}
			if newEntry.Name == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.CreateEntry(&newEntry)

		}
		http.Redirect(w, r, r.Header.Get("/"), 302)
	}
}

func delete(s db.GuestBookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.ParseForm()
		strUuid := r.Form.Get("Delete")
		uuidStr, _ := uuid.Parse(strUuid)

		deleteEntry := model.GuestbookEntry{ID: uuidStr}
		s.DeleteEntry(deleteEntry.ID)
		http.Redirect(w, r, r.Header.Get("/"), 302)

	}
}

func login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("index.html", "header.html", "login.html")
		err := tmplt.Execute(w, nil)
		if err != nil {
			fmt.Println("error when executing template", tmplt)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func signupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplt, _ := template.ParseFiles("index.html", "header.html", "signup.html")
		err := tmplt.Execute(w, nil)
		if err != nil {
			fmt.Println("error when executing template", tmplt)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func loginAuth(u db.UserStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("loginHandler running")
		r.ParseForm()
		email := r.FormValue("email")
		storedpassword, _ := u.GetHash(email)
		if err := bcrypt.CompareHashAndPassword(storedpassword, []byte(r.FormValue("email"))); err != nil {
			fmt.Println("error password is not matching", storedpassword)
			fmt.Println("right hash:")
			return
		}
		fmt.Println("correct Password input", storedpassword)
		//execute xyz
	}
}

func signupAuth(u db.UserStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := u.ValidateUserInput(r.Form)
		if err != nil {
			fmt.Println("user form not valid:", err)
			http.Redirect(w, r, "/signup", 302)
			return
		}
		joinedName := strings.Join([]string{r.FormValue("firstname"), r.FormValue("lastname")}, " ")
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), 14)
		newUser := model.User{Email: r.FormValue("email"), Name: joinedName, Password: hashedpassword}
		u.CreateUser(&newUser)
	}
}
