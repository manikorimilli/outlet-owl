package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
)

// OutletService lists and adds outlets. *outlets.Service satisfies it.
type OutletService interface {
	List(ctx context.Context, user auth.User) ([]outlets.Summary, error)
	Create(ctx context.Context, user auth.User, name string) (outlets.Summary, error)
}

// outletCreate is OutletCreate in api/openapi.yaml.
type outletCreate struct {
	Name string `json:"name"`
}

// outletSummary is OutletSummary in api/openapi.yaml.
type outletSummary struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Managers      []userRef `json:"managers"`
	ReviewCount   int       `json:"review_count"`
	UntaggedCount int       `json:"untagged_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// userRef is UserRef in api/openapi.yaml.
type userRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func newOutletSummary(s outlets.Summary) outletSummary {
	managers := make([]userRef, len(s.Managers)) // [] in JSON, never null
	for i, m := range s.Managers {
		managers[i] = userRef{ID: m.ID, Name: m.Name}
	}
	return outletSummary{
		ID: s.ID, Name: s.Name, Managers: managers,
		ReviewCount: s.ReviewCount, UntaggedCount: s.UntaggedCount, CreatedAt: s.CreatedAt.UTC(),
	}
}

// listOutlets answers GET /api/v1/outlets (operationId listOutlets): every
// outlet for the brand admin, the manager's own outlet for a manager.
func listOutlets(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		list, err := d.Outlets.List(r.Context(), user)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		data := make([]outletSummary, len(list))
		for i, s := range list {
			data[i] = newOutletSummary(s)
		}
		WriteJSON(w, http.StatusOK, map[string]any{"data": data})
	})
}

// createOutlet answers POST /api/v1/outlets (operationId createOutlet). The
// role is checked before the body is read, so a manager gets 403 whatever
// they send (AC-US-01-001-2).
func createOutlet(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		if err := outlets.CheckCanCreate(user); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		var req outletCreate
		if err := decodeJSON(w, r, &req); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		created, err := d.Outlets.Create(r.Context(), user, req.Name)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		w.Header().Set("Location", "/api/v1/outlets/"+strconv.FormatInt(created.ID, 10))
		WriteJSON(w, http.StatusCreated, newOutletSummary(created))
	})
}
