package jsondb

type GuestbookEntry struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}
