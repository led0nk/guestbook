package main

import (
	"flag"
	"fmt"
	"log"

	"guestbook/db/jsondb"

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

	//	newEntry := model.GuestbookEntry{Name: "Peter Müller", Message: "Message"}
	//	bookStorage.CreateEntry(&newEntry)
	entries, _ := bookStorage.ListEntries()
	fmt.Println("list of entries:", entries)

	//placeholder
	m.HandleFunc("/", handlePage)
	m.HandleFunc("/submit", submit)
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

func handlePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)

	tmplt, _ = template.ParseFiles("index.html")

	/*err := tmplt.Execute(w, &entries)
	//fmt.Fprint(w, &entries)
	if err != nil {
		fmt.Println("error when executing template", err)
		return
	}*/

}

// submits guestbook entry (name, message)
func submit(w http.ResponseWriter, r *http.Request) {
	/*var newEntry model.GuestbookEntry
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		t, _ := template.ParseFiles("index.html")
		t.Execute(w, nil)
	} else {}
		r.ParseForm()
		now := time.Now().Format(time.RFC850)
		newEntry = model.GuestbookEntry{ID: uuid.New(), Name: r.FormValue("name"), Message: r.FormValue("message"), CreatedAt: now}
		/*entries = append(entries, newEntry)
		fmt.Print(entries)
		//jsondb.WriteJSON(dbcon, &entries)
	}
	http.Redirect(w, r, r.Header.Get("/"), 302)*/

}
