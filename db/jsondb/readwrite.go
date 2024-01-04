package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"guestbook/model"
	"os"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	CreateEntry(*model.GuestbookEntry) (uuid.UUID, error)
	ListEntries() ([]*model.GuestbookEntry, error)
	DeleteEntry(uuid.UUID) error
}

type BookStorage struct {
	filename string
	entries  map[uuid.UUID]*model.GuestbookEntry
}

func getEntry(s Storage, entries *model.GuestbookEntry) (uuid.UUID, error) {
	return s.CreateEntry(entries)
}

// creates new Storage for entries
func CreateBookStorage(filename string) (*BookStorage, error) {
	storage := &BookStorage{
		filename: filename,
		entries:  make(map[uuid.UUID]*model.GuestbookEntry),
	}
	if err := storage.readJSON(); err != nil {
		return nil, err
	}
	return storage, nil
}

// create new entry in GuestStorage
func (b *BookStorage) CreateEntry(entry *model.GuestbookEntry) (uuid.UUID, error) {

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	b.entries[entry.ID] = entry

	timestamp := time.Now().Format(time.RFC850)
	entry.CreatedAt = timestamp

	if err := b.writeJSON(); err != nil {
		return uuid.Nil, err
	}

	return entry.ID, nil

}

// list entries from Storage
func (b *BookStorage) ListEntries() ([]*model.GuestbookEntry, error) {
	entrylist := make([]*model.GuestbookEntry, 0, len(b.entries))
	for _, entry := range b.entries {
		entrylist = append(entrylist, entry)
	}
	return entrylist, nil
}

// write JSON data into readable format in file = filename
func (b *BookStorage) writeJSON() error {

	//f, _ := os.Create(*filename)
	//defer f.Close()
	as_json, err := json.MarshalIndent(b.entries, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(b.filename, as_json, 0644)
	if err != nil {
		return err
	}
	return nil
}

// read JSON data from file = filename
func (b *BookStorage) readJSON() error {
	if _, err := os.Stat(b.filename); os.IsNotExist(err) {
		fmt.Println("file does not exist", err)
		return nil
	}
	data, err := os.ReadFile(b.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.entries)
}

// delete Entry from storage and write to JSON
func (b *BookStorage) DeleteEntry(entryID uuid.UUID) error {
	if entryID == uuid.Nil {
		return errors.New("requires an entryID")
	}
	if _, exists := b.entries[entryID]; !exists {
		err := errors.New("entry doesn't exist")
		return err
	}

	delete(b.entries, entryID)

	if err := b.writeJSON(); err != nil {
		return err
	}

	return nil
}
