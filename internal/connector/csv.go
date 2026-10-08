package connector

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits from api/openapi.yaml (product owner, 2026-10-06).
const (
	MaxFileBytes  = 5 << 20
	maxTextRunes  = 5000
	maxShortRunes = 200
	// maxQuotedRunes bounds a value quoted back in a rejection reason.
	maxQuotedRunes = 60
)

// Columns are the CSV header names, all required (REQ-003).
var Columns = []string{"outlet", "source", "date", "rating", "text", "reviewer_name"}

// CSV reads one uploaded CSV file. Each row is checked here, the one place a
// CSV row is validated; the import flow adds only the outlet match.
type CSV struct {
	r io.Reader
}

// NewCSV reads from r, which is read once, at most MaxFileBytes.
func NewCSV(r io.Reader) *CSV { return &CSV{r: r} }

// Name implements Connector.
func (*CSV) Name() string { return "csv" }

// Fetch implements Connector.
func (c *CSV) Fetch(ctx context.Context) (Batch, error) {
	data, err := io.ReadAll(&capReader{r: c.r, left: MaxFileBytes})
	if err != nil {
		return Batch{}, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	rd := csv.NewReader(bytes.NewReader(data))
	rd.FieldsPerRecord = -1 // a short row is rejected alone, not the file

	header, err := rd.Read()
	if errors.Is(err, io.EOF) {
		return Batch{}, missingColumns(Columns)
	}
	if err != nil {
		return Batch{}, notCSV()
	}
	index, ferr := columnIndex(header)
	if ferr != nil {
		return Batch{}, ferr
	}

	var b Batch
	for number := 2; ; number++ {
		if err := ctx.Err(); err != nil {
			return Batch{}, err
		}
		record, err := rd.Read()
		if errors.Is(err, io.EOF) {
			return b, nil
		}
		if err != nil {
			return Batch{}, notCSV()
		}
		row, reason := checkRow(number, record, index, len(header))
		if reason != "" {
			b.Rejections = append(b.Rejections, Rejection{Number: number, Reason: reason})
			continue
		}
		b.Rows = append(b.Rows, row)
	}
}

// columnIndex maps each required column to its position. Names are
// compared trimmed and ignoring case; other columns are ignored.
func columnIndex(header []string) (map[string]int, *FileError) {
	index := map[string]int{}
	var repeated []string
	for i, h := range header {
		name := strings.ToLower(strings.TrimSpace(h))
		if _, seen := index[name]; seen {
			repeated = append(repeated, name)
			continue
		}
		index[name] = i
	}
	var missing []string
	for _, c := range Columns {
		if _, ok := index[c]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return nil, missingColumns(missing)
	}
	for _, name := range repeated {
		for _, c := range Columns {
			if name == c {
				return nil, &FileError{
					Message:  fmt.Sprintf("This file has two %s columns, so nothing was imported. Keep one.", name),
					Problems: []Problem{{Field: name, Reason: "repeated"}},
				}
			}
		}
	}
	return index, nil
}

func missingColumns(missing []string) *FileError {
	problems := make([]Problem, len(missing))
	for i, m := range missing {
		problems[i] = Problem{Field: m, Reason: "missing"}
	}
	return &FileError{
		Message: fmt.Sprintf("This file has no %s column, so nothing was imported. The first row must name: %s.",
			missing[0], strings.Join(Columns, ", ")),
		Problems: problems,
	}
}

func notCSV() *FileError {
	return &FileError{
		Message:  "This file could not be read as CSV, so nothing was imported. Export it again as CSV (comma-separated, UTF-8).",
		Problems: []Problem{{Field: "file", Reason: "not_csv"}},
	}
}

// checkRow returns the row, or the reason it is rejected: the first rule it
// breaks, in the order of the phase 3 LLD section 4.
func checkRow(number int, record []string, index map[string]int, width int) (Row, string) {
	if len(record) != width {
		return Row{}, fmt.Sprintf("Row has %d values; the header has %d.", len(record), width)
	}
	for _, v := range record {
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) {
			return Row{}, "Row holds characters that are not UTF-8 text. Export the file again as UTF-8."
		}
	}
	get := func(col string) string { return record[index[col]] }
	blank := func(v string) bool { return strings.TrimSpace(v) == "" }

	row := Row{Number: number, Outlet: get("outlet"), Source: get("source"), Text: get("text"), ReviewerName: get("reviewer_name")}
	switch {
	case blank(row.Outlet):
		return Row{}, "Outlet is empty."
	case blank(row.Source):
		return Row{}, "Source is empty."
	case utf8.RuneCountInString(row.Source) > maxShortRunes:
		return Row{}, "Source is over 200 characters."
	}

	date := get("date")
	if blank(date) {
		return Row{}, "Date is empty."
	}
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(date))
	if err != nil {
		return Row{}, fmt.Sprintf("Date is %q; use YYYY-MM-DD, for example 2026-09-14.", Quote(date))
	}
	row.Date = d

	rating := get("rating")
	if blank(rating) {
		return Row{}, "Rating is empty."
	}
	n, err := strconv.Atoi(strings.TrimSpace(rating))
	if err != nil || n < 1 || n > 5 {
		return Row{}, fmt.Sprintf("Rating is %q; ratings must be a whole number from 1 to 5.", Quote(rating))
	}
	row.Rating = n

	switch {
	case blank(row.Text):
		return Row{}, "Text is empty."
	case utf8.RuneCountInString(row.Text) > maxTextRunes:
		return Row{}, "Text is over 5,000 characters."
	case blank(row.ReviewerName):
		return Row{}, "Reviewer name is empty."
	case utf8.RuneCountInString(row.ReviewerName) > maxShortRunes:
		return Row{}, "Reviewer name is over 200 characters."
	}
	return row, ""
}

// Quote cuts a value quoted back in a reason, so a long cell cannot flood
// the rejected-rows table.
func Quote(v string) string {
	if utf8.RuneCountInString(v) <= maxQuotedRunes {
		return v
	}
	return string([]rune(v)[:maxQuotedRunes]) + "..."
}

// capReader fails with ErrFileTooLarge once more than left bytes are read.
type capReader struct {
	r    io.Reader
	left int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.left < 0 {
		return 0, ErrFileTooLarge
	}
	if int64(len(p)) > c.left+1 {
		p = p[:c.left+1]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	if c.left < 0 {
		return n, ErrFileTooLarge
	}
	return n, err
}
