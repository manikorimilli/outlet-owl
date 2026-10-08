package connector

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const header = "outlet,source,date,rating,text,reviewer_name\n"

func fetch(t *testing.T, csv string) (Batch, error) {
	t.Helper()
	return NewCSV(strings.NewReader(csv)).Fetch(context.Background())
}

// AC-US-01-002-2: the six values are kept unchanged, spaces included.
func TestCSV_ReadsSixColumnsUnchanged(t *testing.T) {
	b, err := fetch(t, "\xef\xbb\xbf"+"Outlet,Source,Date,Rating,Text,Reviewer_Name,extra\n"+
		`Indiranagar,Google,2026-09-14,4,"Great dosa, ""crisp""  ",Asha K.,ignored`+"\n")
	if err != nil || len(b.Rejections) != 0 || len(b.Rows) != 1 {
		t.Fatalf("got %+v, %v; want one row", b, err)
	}
	want := Row{Number: 2, Outlet: "Indiranagar", Source: "Google", Date: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		Rating: 4, Text: `Great dosa, "crisp"  `, ReviewerName: "Asha K."}
	if b.Rows[0] != want {
		t.Fatalf("row = %+v, want %+v", b.Rows[0], want)
	}
}

// AC-US-01-002-3: each rejected row has its row number and one reason; the
// valid rows are still read.
func TestCSV_RejectsRowsWithReasons(t *testing.T) {
	long := strings.Repeat("a", 201)
	cases := []struct{ row, reason string }{
		{"A,G,2026-09-14,4,t", "Row has 5 values; the header has 6."},
		{"A,G,2026-09-14,4,bad \xff byte,R", "Row holds characters that are not UTF-8 text. Export the file again as UTF-8."},
		{"A,G,2026-09-14,4,nul \x00 byte,R", "Row holds characters that are not UTF-8 text. Export the file again as UTF-8."},
		{" ,G,2026-09-14,4,t,R", "Outlet is empty."},
		{"A,,2026-09-14,4,t,R", "Source is empty."},
		{"A," + long + ",2026-09-14,4,t,R", "Source is over 200 characters."},
		{"A,G,,4,t,R", "Date is empty."},
		{"A,G,14/09/2026,4,t,R", `Date is "14/09/2026"; use YYYY-MM-DD, for example 2026-09-14.`},
		{"A,G,2026-09-14,,t,R", "Rating is empty."},
		{"A,G,2026-09-14,4.5,t,R", `Rating is "4.5"; ratings must be a whole number from 1 to 5.`},
		{"A,G,2026-09-14,6,t,R", `Rating is "6"; ratings must be a whole number from 1 to 5.`},
		{"A,G,2026-09-14,4,  ,R", "Text is empty."},
		{"A,G,2026-09-14,4," + strings.Repeat("t", 5001) + ",R", "Text is over 5,000 characters."},
		{"A,G,2026-09-14,4,t,", "Reviewer name is empty."},
		{"A,G,2026-09-14,4,t," + long, "Reviewer name is over 200 characters."},
	}
	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			b, err := fetch(t, header+"A,G,2026-09-14,5,ok,R\n"+tc.row+"\n")
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Rows) != 1 || len(b.Rejections) != 1 {
				t.Fatalf("rows %d rejections %+v; want 1 and 1", len(b.Rows), b.Rejections)
			}
			if got := b.Rejections[0]; got.Number != 3 || got.Reason != tc.reason {
				t.Fatalf("rejection = %+v, want row 3 %q", got, tc.reason)
			}
		})
	}
}

func TestCSV_QuotedValueIsCut(t *testing.T) {
	b, _ := fetch(t, header+"A,G,"+strings.Repeat("9", 80)+",4,t,R\n")
	if r := b.Rejections[0].Reason; !strings.Contains(r, strings.Repeat("9", 60)+`..."`) || strings.Contains(r, strings.Repeat("9", 61)) {
		t.Fatalf("reason %q should quote 60 characters", r)
	}
}

// A quoted value over two lines is one spreadsheet row.
func TestCSV_RowNumbersCountRecordsNotLines(t *testing.T) {
	b, err := fetch(t, header+"A,G,2026-09-14,4,\"two\nlines\",R\nA,G,,4,t,R\n")
	if err != nil || len(b.Rows) != 1 || b.Rows[0].Number != 2 || b.Rejections[0].Number != 3 {
		t.Fatalf("got %+v, %v; want row 2 valid and row 3 rejected", b, err)
	}
}

func TestCSV_FileErrors(t *testing.T) {
	cases := []struct {
		name, csv string
		want      []Problem
	}{
		{"Empty", "", []Problem{{"outlet", "missing"}, {"source", "missing"}, {"date", "missing"}, {"rating", "missing"}, {"text", "missing"}, {"reviewer_name", "missing"}}},
		{"MissingRating", "outlet,source,date,text,reviewer_name\n", []Problem{{"rating", "missing"}}},
		{"RepeatedColumn", "outlet,source,date,rating,text,reviewer_name,Text\n", []Problem{{"text", "repeated"}}},
		{"BareQuote", header + "A,G,2026-09-14,4,he said \"hi,R\n", []Problem{{"file", "not_csv"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fetch(t, tc.csv)
			var fe *FileError
			if !errors.As(err, &fe) {
				t.Fatalf("err = %v, want a FileError", err)
			}
			if len(fe.Problems) != len(tc.want) {
				t.Fatalf("problems = %+v, want %+v", fe.Problems, tc.want)
			}
			for i := range tc.want {
				if fe.Problems[i] != tc.want[i] {
					t.Fatalf("problems = %+v, want %+v", fe.Problems, tc.want)
				}
			}
		})
	}
}

func TestCSV_MissingColumnMessageNamesIt(t *testing.T) {
	_, err := fetch(t, "outlet,source,date,text,reviewer_name\n")
	if !strings.Contains(err.Error(), "no rating column, so nothing was imported") {
		t.Fatalf("message %q should name the rating column", err)
	}
}

func TestCSV_TooLarge(t *testing.T) {
	big := header + strings.Repeat("A,G,2026-09-14,4,"+strings.Repeat("t", 1000)+",R\n", MaxFileBytes/1000)
	if _, err := fetch(t, big); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
	exact := header + strings.Repeat("x", MaxFileBytes-len(header))
	if _, err := fetch(t, exact); errors.Is(err, ErrFileTooLarge) {
		t.Fatal("a file of exactly 5 MB must not be refused for size")
	}
}

// AC-US-01-002-5: the Google connector is declared only and fetches nothing.
func TestGoogle_FetchesNothing(t *testing.T) {
	var c Connector = Google{}
	b, err := c.Fetch(context.Background())
	if !errors.Is(err, ErrNotImplemented) || len(b.Rows) != 0 || c.Name() != "google" {
		t.Fatalf("Fetch = %+v, %v; want nothing and ErrNotImplemented", b, err)
	}
}
