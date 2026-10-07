package gateway

import (
	"context"
	"fmt"
	"time"
)

// reconcileTimeout bounds the key endpoint call at start.
const reconcileTimeout = 10 * time.Second

// Reconciliation is what Reconcile did, for the start-up budget log line.
type Reconciliation struct {
	// State is skipped (replay mode), added, not needed, or unreconciled.
	State    string
	AddedUSD string
	// Err is why the start is unreconciled; the server starts anyway.
	Err error
}

// Reconcile keeps the larger of the local total and the usage OpenRouter
// reports for the key, by inserting the difference when the provider's figure
// is higher (HLD section 3, data model section 4). Only the server calls it,
// once at start. Replay mode builds no HTTP client, so it skips.
func (g *Gateway) Reconcile(ctx context.Context) Reconciliation {
	if g.sender == nil {
		return Reconciliation{State: "skipped"}
	}
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()
	usage, err := g.sender.keyUsage(ctx)
	if err != nil {
		return Reconciliation{State: "unreconciled", Err: err}
	}
	added, ok, err := g.cfg.Store.InsertReconciliation(ctx, usage)
	if err != nil {
		return Reconciliation{State: "unreconciled", Err: fmt.Errorf("record the provider's usage: %w", err)}
	}
	if !ok {
		return Reconciliation{State: "not needed"}
	}
	return Reconciliation{State: "added", AddedUSD: added}
}
