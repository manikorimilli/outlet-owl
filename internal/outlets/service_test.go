package outlets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// memStore is an in-memory Store that applies the same rules as the SQL:
// names unique ignoring case, the scope as a condition, managers by outlet.
type memStore struct {
	outlets  []Outlet
	managers []Manager
	err      error
}

func (m *memStore) ListOutlets(_ context.Context, scope auth.Scope) ([]Outlet, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []Outlet
	for _, o := range m.outlets {
		if scope.All || o.ID == scope.OutletID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (m *memStore) ListActiveManagers(_ context.Context, ids []int64) ([]Manager, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []Manager
	for _, mg := range m.managers {
		for _, id := range ids {
			if mg.OutletID == id {
				out = append(out, mg)
			}
		}
	}
	return out, nil
}

func (m *memStore) CreateOutlet(_ context.Context, name string) (Outlet, bool, error) {
	if m.err != nil {
		return Outlet{}, false, m.err
	}
	for _, o := range m.outlets {
		if strings.EqualFold(o.Name, name) {
			return Outlet{}, false, nil
		}
	}
	o := Outlet{ID: int64(len(m.outlets) + 1), Name: name, CreatedAt: time.Now()}
	m.outlets = append(m.outlets, o)
	return o, true, nil
}

func (m *memStore) OutletByName(_ context.Context, name string) (Outlet, error) {
	for _, o := range m.outlets {
		if strings.EqualFold(o.Name, name) {
			return o, nil
		}
	}
	return Outlet{}, errors.New("not found")
}

var (
	admin   = auth.User{ID: 1, Role: auth.RoleBrandAdmin}
	manager = auth.User{ID: 2, Role: auth.RoleOutletManager, Outlet: &auth.OutletRef{ID: 2, Name: "Indiranagar"}}
)

func seeded() *memStore {
	return &memStore{
		outlets:  []Outlet{{ID: 1, Name: "Koramangala"}, {ID: 2, Name: "Indiranagar"}},
		managers: []Manager{{ID: 2, Name: "Neha Kulkarni", OutletID: 2}},
	}
}

func TestCreate_ManagerRefused(t *testing.T) {
	store := seeded()
	svc := NewService(store)

	_, err := svc.Create(context.Background(), manager, "Electronic City")

	if !errors.Is(err, ErrRoleNotAllowed) {
		t.Fatalf("err = %v, want ErrRoleNotAllowed", err)
	}
	if len(store.outlets) != 2 {
		t.Fatal("a refused manager still created an outlet")
	}
}

func TestCreate_TrimsAndStores(t *testing.T) {
	svc := NewService(seeded())

	got, err := svc.Create(context.Background(), admin, "  Electronic City ")

	if err != nil || got.Name != "Electronic City" || len(got.Managers) != 0 {
		t.Fatalf("Create = %+v, %v; want the trimmed name and no managers", got, err)
	}
}

func TestCreate_NameTakenIgnoringCase(t *testing.T) {
	svc := NewService(seeded())

	_, err := svc.Create(context.Background(), admin, "KORAMANGALA")

	var taken *NameTakenError
	if !errors.As(err, &taken) || taken.Existing != "Koramangala" {
		t.Fatalf("err = %v, want NameTakenError naming Koramangala", err)
	}
}

func TestCreate_StoreFailureIsWrapped(t *testing.T) {
	store := seeded()
	store.err = errors.New("connection refused")

	_, err := NewService(store).Create(context.Background(), admin, "Electronic City")

	var taken *NameTakenError
	if err == nil || errors.As(err, &taken) || errors.Is(err, ErrRoleNotAllowed) {
		t.Fatalf("err = %v, want a wrapped store error", err)
	}
}

func TestValidateName_TrimsAndCountsRunes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr string // the ValidationError reason, "" for valid
	}{
		{name: "trimmed", in: "  Koramangala\t", want: "Koramangala"},
		{name: "200 Devanagari characters pass", in: strings.Repeat("क", 200), want: strings.Repeat("क", 200)},
		{name: "201 characters fail", in: strings.Repeat("क", 201), wantErr: "too_long"},
		{name: "BlankAfterTrimIsRejected", in: " \t ", wantErr: "blank"},
		{name: "empty", in: "", wantErr: "blank"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateName(tc.in)
			if tc.wantErr == "" {
				if err != nil || got != tc.want {
					t.Fatalf("ValidateName = %q, %v; want %q", got, err, tc.want)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != "name" || ve.Reason != tc.wantErr {
				t.Fatalf("err = %v, want name %s", err, tc.wantErr)
			}
		})
	}
}

func TestList_ScopesAndAttachesManagers(t *testing.T) {
	svc := NewService(seeded())

	all, err := svc.List(context.Background(), admin)
	if err != nil || len(all) != 2 {
		t.Fatalf("admin list = %d outlets, %v; want 2", len(all), err)
	}
	own, err := svc.List(context.Background(), manager)
	if err != nil || len(own) != 1 || own[0].Name != "Indiranagar" {
		t.Fatalf("manager list = %+v, %v; want only Indiranagar", own, err)
	}
	if len(own[0].Managers) != 1 || own[0].Managers[0].Name != "Neha Kulkarni" {
		t.Fatalf("managers = %+v, want Neha Kulkarni", own[0].Managers)
	}
}

func TestList_ManagerWithoutOutletSeesNothing(t *testing.T) {
	broken := auth.User{ID: 9, Role: auth.RoleOutletManager}

	got, err := NewService(seeded()).List(context.Background(), broken)

	if err != nil || len(got) != 0 {
		t.Fatalf("List = %+v, %v; want nothing", got, err)
	}
}
