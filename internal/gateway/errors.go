// Package gateway is the only door to the model (tenet 1): it caps
// max_tokens, refuses calls once the recorded total is above USD 8, logs the
// cost of every live attempt in budget.model_calls, and records and replays
// responses (phase 2 LLD).
package gateway

import (
	"errors"
	"fmt"
)

// ErrBudgetExhausted means the recorded running total is above the limit, so
// the call was refused and nothing was sent (AC-US-02-001-4, Q-015).
var ErrBudgetExhausted = errors.New("gateway: the model budget is used up (recorded total above USD 8)")

// ErrProviderCreditExhausted means OpenRouter answered 402: the account's
// credit or the key's limit is used up. Callers treat it like the USD 8 stop.
var ErrProviderCreditExhausted = errors.New("gateway: OpenRouter refused the call for credit (HTTP 402)")

// ErrModelUnavailable means the model gave no usable answer: a timeout, a
// network error, 429 or 5xx after the retries, another 4xx (a missing model
// included, with no fallback) or an unreadable body.
var ErrModelUnavailable = errors.New("gateway: the model did not answer")

// ErrRecordingMissing matches every RecordingMissingError.
var ErrRecordingMissing = errors.New("gateway: no recording for this request")

// ErrRecordingMismatch means a recording file holds a different request than
// its key says: a corrupted or hand-edited file.
var ErrRecordingMismatch = errors.New("gateway: the recording holds a different request")

// ErrRecordingNotSaved means a record-mode call was paid and logged but its
// response could not be written.
var ErrRecordingNotSaved = errors.New("gateway: the recording was not saved")

// RecordingMissingError names the recording replay looked for
// (AC-US-02-003-2). Replay never falls back to a live call (tenet 5).
type RecordingMissingError struct {
	Purpose Purpose
	Key     string
	Path    string
}

func (e *RecordingMissingError) Error() string {
	return fmt.Sprintf("gateway: no recording for %s request %s at %s; record it once, deliberately, with MODEL_GATEWAY_MODE=record",
		e.Purpose, e.Key, e.Path)
}

// Is makes errors.Is(err, ErrRecordingMissing) true.
func (e *RecordingMissingError) Is(target error) bool { return target == ErrRecordingMissing }
