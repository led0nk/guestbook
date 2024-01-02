package jsondb

import (
	"encoding/json"
	"os"

	"github.com/led0nk/guestbook/db"
)

// write JSON data into readable format in file = filename
func WriteJSON(filename *string, entries *[]GuestbookEntry) {

	f, _ := os.Create(*filename)
	defer f.Close()
	as_json, _ := json.MarshalIndent(&entries, "", "\t")
	f.Write(as_json)
}

// read JSON data from file = filename
func ReadJSON(filename *string, entries *[]GuestbookEntry) {
	f, _ := os.ReadFile(*filename)
	json.Unmarshal(f, &entries)
}

func (b *db.BookStorage) CreateEntry(entry *GuestbookEntry)
