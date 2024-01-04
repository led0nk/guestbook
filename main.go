package main

import (
	"flag"
	"fmt"
	"log"

	"guestbook/db/jsondb"
	"guestbook/model"
	"net/http"
	"text/template"
	"time"
)

// var filename string = "./jsondb/entries.json"
var tmplt *template.Template

func main() {
	m := http.NewServeMux()

	var (
		addr = flag.String("addr", ":8080", "server port")
		//database = flag.String("data", "./db/jsondb/entries.json", "link to database")
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
func handlePage(bookStorage *jsondb.BookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)

		tmplt, _ = template.ParseFiles("index.html")
		entries, _ := bookStorage.ListEntries()
		err := tmplt.Execute(w, &entries)
		if err != nil {
			fmt.Println("error when executing template", err)
			return
		}
	}
}

// submits guestbook entry (name, message)
func submit(bookStorage *jsondb.BookStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newEntry model.GuestbookEntry
		fmt.Println("method:", r.Method)
		if r.Method == "GET" {
			t, _ := template.ParseFiles("index.html")
			t.Execute(w, nil)
		} else {
			r.ParseForm()
			newEntry = model.GuestbookEntry{Name: r.FormValue("name"), Message: r.FormValue("message")}
			bookStorage.CreateEntry(&newEntry)
			fmt.Print(&newEntry)
		}

		http.Redirect(w, r, r.Header.Get("/"), 302)
	}
}
