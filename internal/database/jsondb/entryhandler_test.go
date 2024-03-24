package jsondb_test

import (
	"testing"

	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/model"
)

func TestListEntries(t *testing.T) {
	tests := []struct {
		name     string
		input    *model.GuestbookEntry
		expected error
	}{
		{
			name: "Entrylist filled",
			input: &model.GuestbookEntry{
				Name:    "John Doe",
				Message: "this is a test for listing guestbook-entries",
			},
			expected: nil,
		},
		{
			name:     "Empty Entrylist",
			input:    &model.GuestbookEntry{},
			expected: nil,
		},
	}
	filename := "test_entries.json"
	//os.Create(filename)
	//TODO: Implement Write & Read Json-File before Testing / Listing

	storage, err := jsondb.CreateBookStorage(filename)
	if err != nil {
		t.Fatalf("Error creating guestbook storage: %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := storage.ListEntries()
			if (err != nil && test.expected == nil) ||
				(err == nil && test.expected != nil) ||
				(err != nil && test.expected != nil &&
					err.Error() != test.expected.Error()) {
				t.Errorf("Test case %s failed, Expected: %v, Got: %v",
					test.name,
					test.expected,
					err)
			}
		})
	}
}
