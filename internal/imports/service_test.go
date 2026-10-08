package imports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
)

var (
	admin   = auth.User{ID: 1, Role: auth.RoleBrandAdmin}
	manager = auth.User{ID: 2, Role: auth.RoleOutletManager}
)

// memStore keeps imports in memory with the rules the SQL keeps: one import
// per request id, one review per natural key.
type memStore struct {
	byKey   map[string]Result
	outlets map[string]int64
	stored  map[string]bool
	saves   int
}

func newMemStore() *memStore {
	return &memStore{byKey: map[string]Result{}, outlets: map[string]int64{"indiranagar": 1, "hsr layout": 2}, stored: map[string]bool{}}
}

func (m *memStore) ImportByRequestID(_ context.Context, key string) (Result, bool, error) {
	r, ok := m.byKey[key]
	return r, ok, nil
}

func (m *memStore) OutletIDsByName(context.Context) (map[string]int64, error) { return m.outlets, nil }

func (m *memStore) SaveImport(_ context.Context, w Write) (Result, bool, error) {
	m.saves++
	if r, ok := m.byKey[w.RequestID]; ok {
		return r, false, nil
	}
	res := Result{ID: int64(len(m.byKey) + 1), FileName: w.FileName, Rejections: w.Rejections, RejectedCount: len(w.Rejections)}
	for _, r := range w.Reviews {
		k := strings.Join([]string{string(rune(r.OutletID)), r.Source, r.Date.String(), r.ReviewerName, r.Text}, "|")
		if m.stored[k] {
			res.DuplicateCount++
			continue
		}
		m.stored[k] = true
		res.ImportedCount++
	}
	m.byKey[w.RequestID] = res
	return res, true, nil
}

type countSignals struct{ n int }

func (c *countSignals) Signal() { c.n++ }

// fixed is a connector with set rows, counting its fetches.
type fixed struct {
	batch   connector.Batch
	fetches int
}

func (f *fixed) Name() string { return "fixed" }
func (f *fixed) Fetch(context.Context) (connector.Batch, error) {
	f.fetches++
	return f.batch, nil
}

func row(n int, outlet string) connector.Row {
	return connector.Row{Number: n, Outlet: outlet, Source: "Google", Date: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), Rating: 4, Text: "t", ReviewerName: "R"}
}

const key = "0192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b"

func TestImport_ManagerRefused(t *testing.T) {
	st, sig := newMemStore(), &countSignals{}
	src := &fixed{}
	_, err := NewService(st, sig).Import(context.Background(), manager, key, "f.csv", src)
	if !errors.Is(err, ErrRoleNotAllowed) || src.fetches != 0 || st.saves != 0 {
		t.Fatalf("err = %v, fetches %d; want ErrRoleNotAllowed before reading", err, src.fetches)
	}
}

// AC-US-01-002-4: the flow takes any connector, here one that is not CSV.
// AC-US-01-001-1 (second half), AC-US-01-002-3: names match ignoring case
// and spaces; an unknown outlet is rejected with its row number, and the
// rejections come back in row order.
func TestImport_RunsThroughAnyConnectorAndRejectsUnknownOutlet(t *testing.T) {
	st, sig := newMemStore(), &countSignals{}
	src := &fixed{batch: connector.Batch{
		Rows:       []connector.Row{row(2, " INDIRANAGAR "), row(4, "Koramangla"), row(5, "hsr layout")},
		Rejections: []connector.Rejection{{Number: 3, Reason: "Date is empty."}},
	}}

	res, err := NewService(st, sig).Import(context.Background(), admin, key, "f.csv", src)

	if err != nil || res.ImportedCount != 2 || res.RejectedCount != 2 {
		t.Fatalf("result %+v, %v; want 2 imported, 2 rejected", res, err)
	}
	want := []connector.Rejection{{Number: 3, Reason: "Date is empty."},
		{Number: 4, Reason: `Unknown outlet "Koramangla". Check the spelling against your outlet names.`}}
	if len(res.Rejections) != 2 || res.Rejections[0] != want[0] || res.Rejections[1] != want[1] {
		t.Fatalf("rejections = %+v, want %+v", res.Rejections, want)
	}
}

// Tenet 8: a repeat of the key returns the first result without reading the
// source again or signalling again.
func TestImport_RepeatReturnsFirstResultUnread(t *testing.T) {
	st, sig := newMemStore(), &countSignals{}
	svc := NewService(st, sig)
	first, _ := svc.Import(context.Background(), admin, key, "f.csv", &fixed{batch: connector.Batch{Rows: []connector.Row{row(2, "Indiranagar")}}})
	src := &fixed{}

	again, err := svc.Import(context.Background(), admin, key, "other.csv", src)

	if err != nil || again.ID != first.ID || again.FileName != "f.csv" || src.fetches != 0 || sig.n != 1 {
		t.Fatalf("repeat = %+v, %v, fetches %d, signals %d; want the first result, unread", again, err, src.fetches, sig.n)
	}
}

// AC-US-01-002-6: tagging starts on its own after an import that stored
// something, and not after one that stored nothing.
func TestImport_SignalsTaggingOnlyWhenSomethingImported(t *testing.T) {
	st, sig := newMemStore(), &countSignals{}
	svc := NewService(st, sig)
	rows := connector.Batch{Rows: []connector.Row{row(2, "Indiranagar")}}
	if _, err := svc.Import(context.Background(), admin, key, "f.csv", &fixed{batch: rows}); err != nil || sig.n != 1 {
		t.Fatalf("signals = %d, %v; want 1", sig.n, err)
	}
	res, err := svc.Import(context.Background(), admin, "1192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", "f.csv", &fixed{batch: rows})
	if err != nil || res.DuplicateCount != 1 || sig.n != 1 {
		t.Fatalf("re-import %+v, signals %d, %v; want 1 duplicate and no new signal", res, sig.n, err)
	}
}

func TestImport_FileErrorStoresNothing(t *testing.T) {
	st := newMemStore()
	_, err := NewService(st, &countSignals{}).Import(context.Background(), admin, key, "f.csv",
		connector.NewCSV(strings.NewReader("outlet,source\n")))
	var fe *connector.FileError
	if !errors.As(err, &fe) || st.saves != 0 {
		t.Fatalf("err = %v, saves %d; want a FileError and nothing stored", err, st.saves)
	}
}
