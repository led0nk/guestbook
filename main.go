package main

import (
	"fmt"
	"guestbook/jsondb"
	"log"
	"net/http"
	"text/template"
	"time"
)

var filename string = "./jsondb/entries.json"

var entries = []jsondb.GuestbookEntry{
	{
		ID:      1,
		Name:    "Hans Peter",
		Message: "pipapo"},
	{
		ID:      2,
		Name:    "Peter Peter",
		Message: "pepepo"},
	{
		ID:      3,
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
	err := srv.ListenAndServe()
	log.Fatal(err)
}

func handlePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	//fmt.Fprint(w, page)
	http.ServeFile(w, r, "index.html")
	jsondb.ReadJSON(filename, &entries)
	fmt.Fprint(w, &entries)

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
		newEntry = jsondb.GuestbookEntry{ID: 1, Name: r.FormValue("name"), Message: r.FormValue("message")}
		entries = append(entries, newEntry)
		fmt.Print(entries)
		jsondb.WriteJSON(filename, &entries)
	}

}
