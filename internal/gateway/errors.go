// Package gateway is the only door to the model (tenet 1): it caps
// max_tokens, refuses calls once the recorded total is above USD 8, logs the
// cost of every live attempt in budget.model_calls, and records and replays
// responses (phase 2 LLD).
package gateway

import "errors"

// ErrBudgetExhausted means the recorded running total is above the limit, so
// the call was refused and nothing was sent (AC-US-02-001-4, Q-015).
var ErrBudgetExhausted = errors.New("gateway: the model budget is used up (recorded total above USD 8)")
