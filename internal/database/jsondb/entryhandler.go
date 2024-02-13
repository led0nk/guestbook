package jsondb

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/led0nk/guestbook/internal/model"

	"github.com/google/uuid"
)

type BookStorage struct {
	filename string
	entries  map[uuid.UUID]*model.GuestbookEntry
	mu       sync.Mutex
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
	b.mu.Lock()
	defer b.mu.Unlock()
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
	b.mu.Lock()
	defer b.mu.Unlock()
	entrylist := make([]*model.GuestbookEntry, 0, len(b.entries))
	for _, entry := range b.entries {
		entrylist = append(entrylist, entry)
	}

	sort.Slice(entrylist, func(i, j int) bool { return entrylist[i].CreatedAt > entrylist[j].CreatedAt })
	return entrylist, nil

}

// write JSON data into readable format in file = filename
func (b *BookStorage) writeJSON() error {

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
		return errors.New("file does not exist")
	}
	data, err := os.ReadFile(b.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.entries)
}

// delete Entry from storage and write to JSON
func (b *BookStorage) DeleteEntry(entryID uuid.UUID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
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

func (b *BookStorage) GetEntryByName(name string) ([]*model.GuestbookEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if name == "" {
		return nil, errors.New("requires a name")
	}

	entries := []*model.GuestbookEntry{}
	for _, entry := range b.entries {
		if entry.Name == name {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, errors.New("no entries found for " + name)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt > entries[j].CreatedAt })
	return entries, nil
}

func (b *BookStorage) GetEntryByID(id uuid.UUID) ([]*model.GuestbookEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == uuid.Nil {
		return nil, errors.New("requires a uuid")
	}

	entries := []*model.GuestbookEntry{}
	for _, entry := range b.entries {
		if entry.UserID == id {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, errors.New("no entries found for ")

	}
	return entries, nil
}

func (b *BookStorage) GetEntryBySnippet(snippet string) ([]*model.GuestbookEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entries := []*model.GuestbookEntry{}
	for _, entry := range b.entries {
		if strings.Contains(entry.Name, snippet) {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, errors.New("no entries found for " + snippet)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt > entries[j].CreatedAt })
	return entries, nil
}
