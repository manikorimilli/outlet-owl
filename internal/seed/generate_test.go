package seed

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

var monday = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)

// AC-US-02-006-3: exactly 1,500 reviews over 26 weeks ending with the latest
// complete week; AC-US-02-006-4: the planted spike.
func TestGenerate_CountsDatesAndSpike(t *testing.T) {
	rs := Generate(monday)
	if len(rs) != Total {
		t.Fatalf("reviews = %d, want %d", len(rs), Total)
	}
	first := monday.AddDate(0, 0, -7*(Weeks-1))
	waitNow, waitBefore := 0, 0
	seen := map[string]bool{}
	for _, r := range rs {
		if r.Date.Before(first) || r.Date.After(monday.AddDate(0, 0, 6)) {
			t.Fatalf("date %v outside the 26 weeks", r.Date)
		}
		key := r.ReviewerName + r.Text + r.Date.String() + r.Source
		if seen[key] {
			t.Fatalf("duplicate natural key %q", key)
		}
		seen[key] = true
		isWait := slices.Contains(waits, r.Text) || strings.Contains(r.Text, "Waited 40 minutes") || strings.Contains(r.Text, "Bahut wait")
		if r.Outlet == SpikeOutlet && isWait {
			if !r.Date.Before(monday) {
				waitNow++
			} else {
				waitBefore++
			}
		}
	}
	avg := float64(waitBefore) / float64(Weeks-1)
	if float64(waitNow) < 3*avg || waitNow < SpikeExtra {
		t.Fatalf("spike week has %d wait reviews against a weekly average of %.1f", waitNow, avg)
	}
}

// The recordings depend on it: the same week gives the same texts in order.
func TestGenerate_Deterministic(t *testing.T) {
	a, b := Generate(monday), Generate(monday)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two runs differ")
	}
	later := Generate(monday.AddDate(0, 0, 7))
	for i := range a {
		if a[i].Text != later[i].Text || a[i].ReviewerName != later[i].ReviewerName {
			t.Fatalf("review %d changed text with the week", i)
		}
	}
}
