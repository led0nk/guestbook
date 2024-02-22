package v1

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/led0nk/guestbook/internal/model"
)

func testHandlePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handlePage))
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("expected 200 but got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var entrymodel *model.GuestbookEntry
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if body != entrymodel {
		t.Error("expected %s, but we got %s", entrymodel, body)
	}

}
