package httpapi

import (
	"context"
	"net/http"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// StatusReader reads what the tagging status reports: the untagged count in
// the caller's scope, the worker state, the budget and the provider's credit.
type StatusReader interface {
	UntaggedCount(ctx context.Context, scope auth.Scope) (int, error)
	BudgetCents(ctx context.Context) (int64, error)
	WorkerState() string
	CreditExhausted() bool
}

// limitCents is the USD 8.00 stop (REQ-031) in cents.
const limitCents = 800

// getTaggingStatus answers GET /api/v1/tagging/status (operationId
// getTaggingStatus). The untagged count is the caller's scope; the worker
// and budget are the installation's.
func getTaggingStatus(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		untagged, err := d.Status.UntaggedCount(r.Context(), user.Scope())
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		cents, err := d.Status.BudgetCents(r.Context())
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		state := "ok"
		switch {
		case cents > limitCents:
			state = "budget_exhausted"
		case d.Status.CreditExhausted():
			state = "provider_credit_exhausted"
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"untagged_count": untagged,
			"worker":         d.Status.WorkerState(),
			"budget":         map[string]any{"state": state, "spent_minor": cents, "limit_minor": limitCents, "currency": "USD"},
		})
	})
}
