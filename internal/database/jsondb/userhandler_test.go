package jsondb_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/model"
)

func TestValidateUserInput(t *testing.T) {
	tests := []struct {
		name     string
		input    url.Values
		expected error
	}{
		{
			name: "Valid input",
			input: url.Values{"firstname": {"John"}, "lastname": {"Doe"},
				"password": {"password123", "password123"},
				"email":    {"john@doe.com"}},
			expected: nil,
		},
		{
			name: "Empty field",
			input: url.Values{"firstname": {""}, "lastname": {"Doe"},
				"password": {"password123", "password123"},
				"email":    {"john@doe.com"}},
			expected: errors.New("fields cannot be empty"),
		},
		{
			name: "Number in firstname",
			input: url.Values{"firstname": {"John1"}, "lastname": {"Doe"},
				"password": {"password123", "password123"},
				"email":    {"john@doe.com"}},
			expected: errors.New("no numbers allowed"),
		},
		{
			name: "Mismatched password",
			input: url.Values{"firstname": {"John"}, "lastname": {"Doe"},
				"password": {"password123", "password456"},
				"email":    {"john@doe.com"}},
			expected: errors.New("password doesn't match, please try again"),
		},
		{
			name: "Long password",
			input: url.Values{"firstname": {"John"}, "lastname": {"Doe"},
				"password": {"averylongpasswordthatexceedsseventytwocharactersandistoolongforthistestcase", "averylongpasswordthatexceedsseventytwocharactersandistoolongforthistestcase"},
				"email":    {"john@doe.com"}},
			expected: errors.New("password is too long, only 72 characters allowed"),
		},
		{
			name: "Short password",
			input: url.Values{"firstname": {"John"}, "lastname": {"Doe"},
				"password": {"short", "short"},
				"email":    {"john@doe.com"}},
			expected: errors.New("password is too short, should be at least 8 characters long"),
		},
		{
			name: "Email format",
			input: url.Values{"firstname": {"John"}, "lastname": {"Doe"},
				"password": {"password123", "password123"},
				"email":    {"johndoe.com"}},
			expected: errors.New("email is not in correct format, please try again"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := jsondb.ValidateUserInput(test.input)
			if (err != nil && test.expected == nil) ||
				(err == nil && test.expected != nil) ||
				(err != nil && test.expected != nil &&
					err.Error() != test.expected.Error()) {
				t.Errorf("Test case %s failed. Expected: %v, Got: %v",
					test.name,
					test.expected,
					err)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	filename := "test_users.json"

	//testUser := &model.User{
	//	Name:  "Jon Doe",
	//	Email: "jon@doe.com",
	//}

	//testUserJSON, err := json.MarshalIndent(testUser, "", " \t")
	//if err != nil {
	//	t.Fatalf("Error marshaling user JSON: %v", err)
	//}
	//err = os.WriteFile(filename, testUserJSON, 0644)
	//if err != nil {
	//	t.Fatalf("Error writing test user JSON to file: %v", err)
	//}
	defer os.Remove(filename)

	storage, err := jsondb.CreateUserStorage(filename)
	if err != nil {
		t.Fatalf("Error creating user storage: %v", err)
	}

	// Create a sample user
	user := &model.User{
		Name: "Test User",
	}

	// Create the user
	id, err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("Error creating user: %v", err)
	}

	// Verify user ID is not nil
	if id == uuid.Nil {
		t.Errorf("Expected non-nil UUID for user, got nil")
	}

	// Read the data back from file
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	// Unmarshal the data to verify correctness
	var users map[uuid.UUID]*model.User
	err = json.Unmarshal(data, &users)
	if err != nil {
		t.Fatalf("Error unmarshaling data: %v", err)
	}

	// Verify the user was written correctly
	if len(users) != 1 {
		t.Errorf("Expected 1 user in storage, got %d", len(users))
	}

	// Verify user ID in storage matches created user ID
	if _, ok := users[id]; !ok {
		t.Errorf("Expected user with ID %s in storage, not found", id)
	}
}
