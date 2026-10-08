package reviews

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// Cursor is the keyset position after the last review of a page: its date
// and id, in the list's order (review_date, id) descending.
type Cursor struct {
	Date time.Time
	ID   int64
}

type cursorJSON struct {
	D string `json:"d"`
	I int64  `json:"i"`
}

// ErrBadCursor means the cursor did not come from this API (400).
var ErrBadCursor = errors.New("reviews: the cursor is not valid")

// Encode gives the opaque next_cursor.
func (c Cursor) Encode() string {
	b, _ := json.Marshal(cursorJSON{D: c.Date.Format(time.DateOnly), I: c.ID}) // two plain fields always marshal
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor reads a next_cursor back.
func DecodeCursor(s string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}
	var c cursorJSON
	if err := json.Unmarshal(b, &c); err != nil || c.I < 1 {
		return Cursor{}, ErrBadCursor
	}
	d, err := time.Parse(time.DateOnly, c.D)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}
	return Cursor{Date: d, ID: c.I}, nil
}
