package reviews

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

type memStore struct {
	rows  []Review
	calls int
}

func (m *memStore) ListReviews(_ context.Context, _ auth.Scope, _ Filter, limit int) ([]Review, error) {
	m.calls++
	if limit < len(m.rows) {
		return m.rows[:limit], nil
	}
	return m.rows, nil
}
func (m *memStore) CountReviews(context.Context, auth.Scope, Filter) (int, error) {
	return len(m.rows), nil
}

var (
	admin   = auth.User{Role: auth.RoleBrandAdmin}
	manager = auth.User{Role: auth.RoleOutletManager, Outlet: &auth.OutletRef{ID: 3}}
)

func TestList_UnknownThemeIs422(t *testing.T) {
	var v *ValidationError
	if _, err := NewService(&memStore{}).List(context.Background(), admin, Filter{Theme: "ambience"}); !errors.As(err, &v) || v.Field != "filter[theme]" {
		t.Fatalf("err = %v, want a filter[theme] ValidationError", err)
	}
}

// AC-US-00-001-4: a manager asking for another outlet's reviews is refused.
func TestList_ManagerOtherOutletIs404(t *testing.T) {
	other, own := int64(4), int64(3)
	st := &memStore{}
	if _, err := NewService(st).List(context.Background(), manager, Filter{OutletID: &other}); !errors.Is(err, ErrNotFound) || st.calls != 0 {
		t.Fatalf("err = %v, calls %d; want ErrNotFound before any query", err, st.calls)
	}
	if _, err := NewService(st).List(context.Background(), manager, Filter{OutletID: &own}); err != nil {
		t.Fatalf("own outlet: %v", err)
	}
}

func TestList_ReplyStatusBeforePhase5(t *testing.T) {
	st := &memStore{rows: []Review{{ID: 1}}}
	p, err := NewService(st).List(context.Background(), admin, Filter{ReplyStatus: "replied"})
	if err != nil || len(p.Reviews) != 0 || p.Total != 0 || st.calls != 0 {
		t.Fatalf("page %+v, %v; want nothing", p, err)
	}
}

func TestList_PagesWithACursor(t *testing.T) {
	day := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	st := &memStore{rows: []Review{{ID: 9, ReviewDate: day}, {ID: 8, ReviewDate: day}, {ID: 7, ReviewDate: day}}}
	p, err := NewService(st).List(context.Background(), admin, Filter{Limit: 2})
	if err != nil || len(p.Reviews) != 2 || p.Next == nil || p.Next.ID != 8 || p.Total != 3 {
		t.Fatalf("page %+v, %v; want 2 rows and a cursor after 8", p, err)
	}
}

func TestCursor_RoundTripAndRefusesJunk(t *testing.T) {
	c := Cursor{Date: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), ID: 1497}
	back, err := DecodeCursor(c.Encode())
	if err != nil || back != c {
		t.Fatalf("round trip = %+v, %v", back, err)
	}
	for _, junk := range []string{"x", "e30", "eyJkIjoieCIsImkiOjF9"} {
		if _, err := DecodeCursor(junk); err == nil {
			t.Errorf("%q decoded", junk)
		}
	}
}

func TestLikePattern_EscapesWildcards(t *testing.T) {
	if got := LikePattern(`50%_off\`); got != `%50\%\_off\\%` {
		t.Fatalf("pattern = %q", got)
	}
}
