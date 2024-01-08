package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"github.com/led0nk/guestbook/db"
	"github.com/led0nk/guestbook/db/jsondb"
	"github.com/led0nk/guestbook/model"

	"github.com/google/uuid"
)

// var filename string = "./jsondb/entries.json"
var tmplt *template.Template

func main() {
	m := http.NewServeMux()

	var (
		addr    = flag.String("addr", ":8080", "server port")
		connStr = flag.String("data", "file://entries.json", "link to database")
	)
	flag.Parse()

	u, err := url.Parse(*connStr)
	if err != nil {
		panic(err)
	}
	log.Print(u)
	var database db.Storage
	switch u.Scheme {
	case "file":
		log.Println("opening:", u.Hostname())
		bookStorage, _ := jsondb.CreateBookStorage("./entries.json")
		database = bookStorage
	default:
		panic("bad storage")
	}

	//placeholder
	m.HandleFunc("/", handlePage(database))
	m.HandleFunc("/submit", submit(database))
	m.HandleFunc("/delete", delete(database))
	m.HandleFunc("/login", login())
	//m.HandleFunc("/signup")
	//m.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

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
func handlePage(s db.Storage) http.HandlerFunc {
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
func submit(s db.Storage) http.HandlerFunc {
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

func delete(s db.Storage) http.HandlerFunc {
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
		err := tmplt.Execute(w, "test")
		if err != nil {
			fmt.Println("error when executing template", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}
