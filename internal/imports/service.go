// Package imports holds the import flow: who may import, the repeat of an
// upload, the outlet match and the counts (US-01-002, phase 3 LLD section
// 5). It reads rows through any connector and has no SQL and no HTTP types.
package imports

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
)

// ErrRoleNotAllowed means the caller is not the brand admin, the only role
// that imports (HLD section 9).
var ErrRoleNotAllowed = errors.New("imports: only the brand admin can import reviews")

// Result is one import as the API returns it (ImportResult).
type Result struct {
	ID             int64
	FileName       string
	ImportedCount  int
	DuplicateCount int
	RejectedCount  int
	Rejections     []connector.Rejection
	CreatedAt      time.Time
}

// NewReview is a valid row matched to its outlet.
type NewReview struct {
	OutletID int64
	connector.Row
}

// Write is everything one import stores, in one transaction.
type Write struct {
	RequestID  string
	FileName   string
	Reviews    []NewReview
	Rejections []connector.Rejection
}

// Store reads and writes imports. SaveImport reports created false when an
// import with the request id already exists, and writes nothing then.
type Store interface {
	ImportByRequestID(ctx context.Context, requestID string) (Result, bool, error)
	OutletIDsByName(ctx context.Context) (map[string]int64, error)
	SaveImport(ctx context.Context, w Write) (result Result, created bool, err error)
}

// Notifier starts tagging after an import commits (Q-022); the tagging
// worker satisfies it.
type Notifier interface {
	Signal()
}

// Service applies the import rules.
type Service struct {
	store  Store
	notify Notifier
}

// NewService builds the import service.
func NewService(store Store, notify Notifier) *Service {
	return &Service{store: store, notify: notify}
}

// CheckCanImport refuses every role but the brand admin. The handler calls
// it before reading the body.
func CheckCanImport(user auth.User) error {
	if user.Role != auth.RoleBrandAdmin {
		return ErrRoleNotAllowed
	}
	return nil
}

// Import reads the connector's rows and stores the valid new ones under the
// request id. A repeat of the request id returns the first result without
// reading the source (tenet 8). Tagging is signalled when something new was
// stored (AC-US-01-002-6).
func (s *Service) Import(ctx context.Context, user auth.User, requestID, fileName string, src connector.Connector) (Result, error) {
	if err := CheckCanImport(user); err != nil {
		return Result{}, err
	}
	if prev, ok, err := s.store.ImportByRequestID(ctx, requestID); err != nil {
		return Result{}, fmt.Errorf("look up import %s: %w", requestID, err)
	} else if ok {
		return prev, nil
	}

	batch, err := src.Fetch(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("read the %s source: %w", src.Name(), err)
	}
	outlets, err := s.store.OutletIDsByName(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("read outlets: %w", err)
	}

	w := Write{RequestID: requestID, FileName: fileName}
	rejected := map[int]string{}
	for _, r := range batch.Rejections {
		rejected[r.Number] = r.Reason
	}
	for _, row := range batch.Rows {
		id, ok := outlets[OutletKey(row.Outlet)]
		if !ok {
			rejected[row.Number] = fmt.Sprintf("Unknown outlet %q. Check the spelling against your outlet names.", connector.Quote(row.Outlet))
			continue
		}
		w.Reviews = append(w.Reviews, NewReview{OutletID: id, Row: row})
	}
	w.Rejections = inRowOrder(rejected)

	res, created, err := s.store.SaveImport(ctx, w)
	if err != nil {
		return Result{}, fmt.Errorf("save import %s: %w", requestID, err)
	}
	if created && res.ImportedCount > 0 {
		s.notify.Signal()
	}
	return res, nil
}

// OutletKey is how a CSV outlet name meets an outlet name: trimmed and
// ignoring case (HLD flow A). The store builds its map with it.
func OutletKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// inRowOrder lists the rejections by row number, the order the API returns.
func inRowOrder(rejected map[int]string) []connector.Rejection {
	out := make([]connector.Rejection, 0, len(rejected))
	for n, reason := range rejected {
		out = append(out, connector.Rejection{Number: n, Reason: reason})
	}
	slices.SortFunc(out, func(a, b connector.Rejection) int { return a.Number - b.Number })
	return out
}
