package jsondb

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/led0nk/guestbook/internal/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
)

var tracer = otel.GetTracerProvider().Tracer("github.com/led0nk/guestbook/intern/db/jsondb")

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
func (b *BookStorage) CreateEntry(ctx context.Context, entry *model.GuestbookEntry) (uuid.UUID, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "CreateEntry")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
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
func (b *BookStorage) ListEntries(ctx context.Context) ([]*model.GuestbookEntry, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "ListEntries")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
	defer b.mu.Unlock()

	span.AddEvent("create list")
	entrylist := make([]*model.GuestbookEntry, 0, len(b.entries))
	for _, entry := range b.entries {
		entrylist = append(entrylist, entry)
	}

	span.AddEvent("sort list")
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
func (b *BookStorage) DeleteEntry(ctx context.Context, entryID uuid.UUID) error {
	var span trace.Span
	_, span = tracer.Start(ctx, "DeleteEntry")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
	defer b.mu.Unlock()

	if entryID == uuid.Nil {
		return errors.New("requires an entryID")
	}
	if _, exists := b.entries[entryID]; !exists {
		err := errors.New("entry doesn't exist")
		return err
	}

	span.AddEvent("delete entry from entrylist")
	delete(b.entries, entryID)

	if err := b.writeJSON(); err != nil {
		return err
	}

	return nil
}

func (b *BookStorage) GetEntryByName(ctx context.Context, name string) ([]*model.GuestbookEntry, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetEntryByName")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
	defer b.mu.Unlock()

	if name == "" {
		return nil, errors.New("requires a name")
	}

	entries := []*model.GuestbookEntry{}
	span.AddEvent("create entry slice")
	for _, entry := range b.entries {
		if entry.Name == name {
			entries = append(entries, entry)
		}
	}

	span.AddEvent("sort slice")
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt > entries[j].CreatedAt })
	return entries, nil
}

func (b *BookStorage) GetEntryByID(ctx context.Context, id uuid.UUID) ([]*model.GuestbookEntry, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetEntryByID")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
	defer b.mu.Unlock()

	if id == uuid.Nil {
		return nil, errors.New("requires a uuid")
	}

	entries := []*model.GuestbookEntry{}
	span.AddEvent("create slice by entryID")
	for _, entry := range b.entries {
		if entry.UserID == id {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (b *BookStorage) GetEntryBySnippet(ctx context.Context, snippet string) ([]*model.GuestbookEntry, error) {
	var span trace.Span
	_, span = tracer.Start(ctx, "GetEntryBySnippet")
	defer span.End()

	span.AddEvent("Lock")
	b.mu.Lock()
	defer span.AddEvent("Unlock")
	defer b.mu.Unlock()

	entries := []*model.GuestbookEntry{}
	span.AddEvent("check entries for snippet")
	for _, entry := range b.entries {
		if strings.Contains(entry.Name, snippet) {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, errors.New("no entries found for " + snippet)
	}

	span.AddEvent("sort entry slice")
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt > entries[j].CreatedAt })
	return entries, nil
}
