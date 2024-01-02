package main

import (
	"fmt"
	"guestbook/jsondb"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/google/uuid"
)

var filename string = "./jsondb/entries.json"
var tmplt *template.Template

var entries = []jsondb.GuestbookEntry{
	{
		ID:      uuid.New(),
		Name:    "Hans Peter",
		Message: "pipapo",
	},
	{
		ID:      uuid.New(),
		Name:    "Peter Peter",
		Message: "pepepo"},
	{
		ID:      uuid.New(),
		Name:    "Hansebanger",
		Message: "bangbangpo"},
}

func main() {
	m := http.NewServeMux()

	const addr = ":8080"

	m.HandleFunc("/", handlePage)
	m.HandleFunc("/submit", submit)

	srv := http.Server{
		Handler:      m,
		Addr:         addr,
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

	jsondb.ReadJSON(filename, &entries)
	tmplt, _ = template.ParseFiles("index.html")

	err := tmplt.Execute(w, &entries)
	//fmt.Fprint(w, &entries)
	if err != nil {
		fmt.Println("error when executing template", err)
		return
	}

}

// submits guestbook entry (name, message)
func submit(w http.ResponseWriter, r *http.Request) {
	var newEntry jsondb.GuestbookEntry
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		t, _ := template.ParseFiles("index.html")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		now := time.Now().Format(time.RFC850)
		newEntry = jsondb.GuestbookEntry{ID: uuid.New(), Name: r.FormValue("name"), Message: r.FormValue("message"), CreatedAt: now}
		entries = append(entries, newEntry)
		fmt.Print(entries)
		jsondb.WriteJSON(filename, &entries)
	}
	http.Redirect(w, r, r.Header.Get("/"), 302)

}
