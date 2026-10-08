// Package connector reads reviews from a source. The import flow takes any
// Connector, so a new source implements the interface without changing the
// flow (REQ-004, AC-US-01-002-4). CSV is the only working connector; Google
// is declared and fetches nothing (REQ-005). Phase 3 LLD sections 3 and 4.
package connector

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Connector is one source of reviews.
type Connector interface {
	// Name identifies the source in logs: csv, google.
	Name() string
	// Fetch returns every row the source holds, each either valid or
	// rejected with its reason. A problem with the whole source is an error,
	// and nothing from it is imported.
	Fetch(ctx context.Context) (Batch, error)
}

// Batch is what a connector read: valid rows and rejected rows, each with
// the row number the admin can find.
type Batch struct {
	Rows       []Row
	Rejections []Rejection
}

// Row is one review as the source holds it, values unchanged
// (AC-US-01-002-2). Outlet is the outlet's name; the import flow matches it.
type Row struct {
	Number       int
	Outlet       string
	Source       string
	Date         time.Time // a calendar date, at midnight UTC
	Rating       int
	Text         string
	ReviewerName string
}

// Rejection is a row that breaks a rule, with a reason worded so the admin
// can fix it (Q-023).
type Rejection struct {
	Number int
	Reason string
}

// FileError is a problem with the whole source: unreadable, or a required
// column missing or repeated. Nothing is imported (API 400 csv_invalid).
type FileError struct {
	Message string
	// Problems names each column and what is wrong with it: missing,
	// repeated, or "file" for a file that is not CSV.
	Problems []Problem
}

// Problem is one entry of a FileError.
type Problem struct {
	Field  string
	Reason string
}

func (e *FileError) Error() string { return "connector: " + e.Message }

// ErrFileTooLarge means the source is over the 5 MB the API accepts.
var ErrFileTooLarge = errors.New("connector: the file is over 5 MB")

// ErrNotImplemented is what a declared-only connector returns.
var ErrNotImplemented = errors.New("connector: not implemented")

// Google is the Google Business Profile connector, declared only: it holds
// no client and makes no network call (REQ-005, AC-US-01-002-5).
type Google struct{}

// Name implements Connector.
func (Google) Name() string { return "google" }

// Fetch implements Connector and always fails.
func (Google) Fetch(context.Context) (Batch, error) {
	return Batch{}, fmt.Errorf("google connector: %w", ErrNotImplemented)
}
