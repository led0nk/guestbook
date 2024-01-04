package main

import (
	"flag"
	"fmt"
	"guestbook/db/jsondb"
	"guestbook/model"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// var filename string = "./jsondb/entries.json"
var tmplt *template.Template

func main() {
	m := http.NewServeMux()

	var (
		addr = flag.String("addr", ":8080", "server port")
		//database = flag.String("data", "./entries.json", "link to database")
	)
	flag.Parse()

	//placeholder
	bookStorage, _ := jsondb.CreateBookStorage("./entries.json")

	//placeholder
	m.HandleFunc("/", handlePage(bookStorage))
	m.HandleFunc("/submit", submit(bookStorage))
	m.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	srv := http.Server{
		Handler:      m,
		Addr:         *addr,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	fmt.Println("server started on port", addr)
	fmt.Println(time.Now())
	err := srv.ListenAndServe()
	log.Fatal(err)
}

// hands over Entries to Handler and prints them out in template
func handlePage(s jsondb.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		tmplt, _ = template.ParseFiles("index.html")
		entries, _ := s.ListEntries()
		err := tmplt.Execute(w, &entries)
		if err != nil {
			fmt.Println("error when executing template", err)
			return
		}
	}
}

func test(s jsondb.Storage, entry *model.GuestbookEntry) (uuid.UUID, error) {
	return s.CreateEntry(entry)
}

// submits guestbook entry (name, message)
func submit(s jsondb.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("method:", r.Method)
		if r.Method == "GET" {
			tmplt, _ := template.ParseFiles("index.html")
			tmplt.Execute(w, nil)
		} else {
			r.ParseForm()
			newEntry := model.GuestbookEntry{Name: r.FormValue("name"), Message: r.FormValue("message")}
			s.CreateEntry(&newEntry)

		}

		http.Redirect(w, r, r.Header.Get("/"), 302)
	}
}
