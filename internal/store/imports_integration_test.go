//go:build integration

package store

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/imports"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

func newImportStore(t *testing.T) (*Store, context.Context, int64) {
	t.Helper()
	db := storetest.New(t)
	s := &Store{Pool: db.Pool}
	ctx := context.Background()
	o, _, err := s.CreateOutlet(ctx, "Indiranagar")
	if err != nil {
		t.Fatal(err)
	}
	return s, ctx, o.ID
}

func review(outlet int64, n int, text string) imports.NewReview {
	return imports.NewReview{OutletID: outlet, Row: connector.Row{Number: n, Outlet: "Indiranagar", Source: "Google",
		Date: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), Rating: 4, Text: text, ReviewerName: "Asha"}}
}

// TestMigration00002_UpDownUp takes migration 2 down and up again; the
// tables and the sentiment type follow it.
func TestMigration00002_UpDownUp(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()
	check := func(want bool) {
		t.Helper()
		for _, name := range []string{"public.imports", "public.import_rejections", "public.reviews", "public.review_tags"} {
			var ok bool
			if err := db.Pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", name).Scan(&ok); err != nil || ok != want {
				t.Fatalf("%s exists = %v (%v), want %v", name, ok, err, want)
			}
		}
		var ok bool
		if err := db.Pool.QueryRow(ctx, "SELECT to_regtype('public.sentiment') IS NOT NULL").Scan(&ok); err != nil || ok != want {
			t.Fatalf("sentiment exists = %v (%v), want %v", ok, err, want)
		}
	}
	check(true)
	db.Goose(t, "down-to", "1")
	check(false)
	db.Goose(t, "up")
	check(true)
}

// AC-US-01-002-1, -2: valid rows become reviews holding the values
// unchanged; a second import of the same rows stores nothing and counts them
// as duplicates; a repeat of the key returns the first result.
func TestImport_StoresRowsAndCountsDuplicates(t *testing.T) {
	s, ctx, outlet := newImportStore(t)
	w := imports.Write{RequestID: "0192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "sept.csv",
		Reviews:    []imports.NewReview{review(outlet, 2, "Great dosa  "), review(outlet, 3, "Slow"), review(outlet, 4, "Slow")},
		Rejections: []connector.Rejection{{Number: 5, Reason: "Date is empty."}}}

	res, created, err := s.SaveImport(ctx, w)
	if err != nil || !created || res.ImportedCount != 2 || res.DuplicateCount != 1 || res.RejectedCount != 1 {
		t.Fatalf("first = %+v, %v, %v; want 2 imported, 1 duplicate (row 4 repeats row 3), 1 rejected", res, created, err)
	}
	var text, source, name string
	var rating int
	var date time.Time
	if err := s.Pool.QueryRow(ctx, "SELECT review_text, source, reviewer_name, rating, review_date FROM reviews ORDER BY id LIMIT 1").
		Scan(&text, &source, &name, &rating, &date); err != nil || text != "Great dosa  " || source != "Google" || name != "Asha" ||
		rating != 4 || !date.Equal(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("stored %q %q %q %d %v (%v); want the row unchanged", text, source, name, rating, date, err)
	}

	again, created, err := s.SaveImport(ctx, w)
	if err != nil || created || again.ID != res.ID || len(again.Rejections) != 1 {
		t.Fatalf("repeat = %+v, %v, %v; want the first result", again, created, err)
	}
	w.RequestID = "1192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b"
	second, _, err := s.SaveImport(ctx, w)
	if err != nil || second.ImportedCount != 0 || second.DuplicateCount != 3 {
		t.Fatalf("re-import = %+v, %v; want 0 imported, 3 duplicates", second, err)
	}
	got, ok, err := s.ImportByRequestID(ctx, w.RequestID)
	if err != nil || !ok || got.DuplicateCount != 3 || got.FileName != "sept.csv" {
		t.Fatalf("stored result = %+v, %v, %v", got, ok, err)
	}
}

// Tenet 8: two uploads with one key at the same moment import once; the
// second waits on the first and returns its result.
func TestImport_ConcurrentSameKeyImportsOnce(t *testing.T) {
	s, ctx, outlet := newImportStore(t)
	w := imports.Write{RequestID: "2192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "f.csv",
		Reviews: []imports.NewReview{review(outlet, 2, "a"), review(outlet, 3, "b")}}
	var wg sync.WaitGroup
	results := make([]imports.Result, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() { results[i], _, errs[i] = s.SaveImport(ctx, w) })
	}
	wg.Wait()
	var reviews int
	_ = s.Pool.QueryRow(ctx, "SELECT count(*) FROM reviews").Scan(&reviews)
	if errs[0] != nil || errs[1] != nil || results[0].ID != results[1].ID || results[1].ImportedCount != 2 || reviews != 2 {
		t.Fatalf("results %+v errs %v reviews %d; want one import of 2 reviews", results, errs, reviews)
	}
}

// S-06 counts: reviews and untagged reviews per outlet.
func TestOutlets_CountReviewsAndUntagged(t *testing.T) {
	s, ctx, outlet := newImportStore(t)
	if _, _, err := s.SaveImport(ctx, imports.Write{RequestID: "3192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "f.csv",
		Reviews: []imports.NewReview{review(outlet, 2, "a"), review(outlet, 3, "b")}}); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.UntaggedReviewIDs(ctx, 0)
	if _, err := s.SaveResults(ctx, []tagging.Result{{ReviewID: ids[0], Themes: []string{}, Sentiment: "positive", UrgentReasons: []string{}}}, 1); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListOutlets(ctx, auth.Scope{All: true})
	if err != nil || len(list) != 1 || list[0].ReviewCount != 2 || list[0].UntaggedCount != 1 {
		t.Fatalf("outlets = %+v, %v; want 2 reviews, 1 untagged", list, err)
	}
}

// AC-US-01-003-6, tenet 4: a stored result is never replaced, and the row
// keeps its prompt version and reasons.
func TestTagging_InsertKeepsTheFirstResult(t *testing.T) {
	s, ctx, outlet := newImportStore(t)
	if _, _, err := s.SaveImport(ctx, imports.Write{RequestID: "4192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "f.csv",
		Reviews: []imports.NewReview{review(outlet, 2, "a")}}); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.UntaggedReviewIDs(ctx, 0)
	first := tagging.Result{ReviewID: ids[0], Themes: []string{"food"}, Sentiment: "negative", UrgentReasons: []string{"food_safety", "legal_threat"}}
	if n, err := s.SaveResults(ctx, []tagging.Result{first}, 1); err != nil || n != 1 {
		t.Fatalf("first save = %d, %v", n, err)
	}
	if n, err := s.SaveResults(ctx, []tagging.Result{{ReviewID: ids[0], Themes: []string{}, Sentiment: "positive", UrgentReasons: []string{}}}, 2); err != nil || n != 0 {
		t.Fatalf("second save = %d, %v; want nothing stored", n, err)
	}
	var urgent bool
	var reasons []string
	var version int
	if err := s.Pool.QueryRow(ctx, "SELECT is_urgent, urgent_reasons, prompt_version FROM review_tags WHERE review_id = $1", ids[0]).
		Scan(&urgent, &reasons, &version); err != nil || !urgent || !slices.Equal(reasons, first.UrgentReasons) || version != 1 {
		t.Fatalf("stored %v %v %d (%v); want the first result", urgent, reasons, version, err)
	}
	if left, _ := s.UntaggedReviewIDs(ctx, 0); len(left) != 0 {
		t.Fatalf("untagged = %v, want none", left)
	}
}

// ADR-0006: while one pass holds the lock a second holder waits; closing the
// first holder's connection lets it in.
func TestTagging_LockExcludesASecondHolder(t *testing.T) {
	s, ctx, _ := newImportStore(t)
	unlock, err := s.LockTagging(ctx)
	if err != nil {
		t.Fatal(err)
	}
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	if _, err := s.LockTagging(short); err == nil || !errors.Is(short.Err(), context.DeadlineExceeded) {
		t.Fatalf("second lock = %v; want it to wait until the deadline", err)
	}
	unlock()
	unlock2, err := s.LockTagging(ctx)
	if err != nil {
		t.Fatalf("lock after unlock: %v", err)
	}
	unlock2()
}
