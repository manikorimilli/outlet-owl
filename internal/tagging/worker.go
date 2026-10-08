package tagging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// Numbers from the HLD section 8 and REQ-006.
const (
	// BatchSize is how many reviews one tagging request holds.
	BatchSize = 20
	// MaxRetries is how many times missing and invalid ids are sent again
	// within one batch (Q-014, AC-US-01-004-5).
	MaxRetries = 2
	// LockKey is the PostgreSQL advisory lock one tagging pass holds, so the
	// phase 7 seed and eval commands never tag at the same time (ADR-0006).
	LockKey int64 = 7300301
)

// Model is the one door to the model (tenet 1); *gateway.Gateway is the
// only production value.
type Model interface {
	Complete(ctx context.Context, r gateway.Request) (gateway.Response, error)
}

// Store is what a pass reads and writes. LockTagging blocks until the
// advisory lock is held on a connection of its own; unlock closes that
// connection, which releases the lock even if nothing else runs.
type Store interface {
	LockTagging(ctx context.Context) (unlock func(), err error)
	UntaggedReviewIDs(ctx context.Context, afterID int64) ([]int64, error)
	ReviewTexts(ctx context.Context, ids []int64) ([]Review, error)
	// SaveResults stores each result unless the review already has one, in
	// one transaction, and returns how many it stored.
	SaveResults(ctx context.Context, results []Result, promptVersion int) (int, error)
}

// Config sets up a worker.
type Config struct {
	Store  Store
	Model  Model
	Prompt prompts.Version
	// Enabled is the operator switch TAGGING_ENABLED (HLD section 12).
	Enabled bool
	Logger  *slog.Logger
}

// Worker is the one tagging goroutine of the server (ADR-0006).
type Worker struct {
	cfg     Config
	signal  chan struct{}
	running atomic.Bool
}

// State is paused, running or idle (TaggingStatus.worker in the API).
func (w *Worker) State() string {
	switch {
	case !w.cfg.Enabled:
		return "paused"
	case w.running.Load():
		return "running"
	}
	return "idle"
}

// New builds a worker; Run starts it.
func New(cfg Config) *Worker {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}
	return &Worker{cfg: cfg, signal: make(chan struct{}, 1)}
}

// Signal asks for a pass and never blocks. Signals that arrive during a pass
// collapse into one more pass, so none is lost (HLD section 3).
func (w *Worker) Signal() {
	select {
	case w.signal <- struct{}{}:
	default:
	}
}

// Run makes one pass per signal until ctx ends. With tagging switched off it
// tags nothing.
func (w *Worker) Run(ctx context.Context) {
	if !w.cfg.Enabled {
		w.cfg.Logger.Info("tagging paused: TAGGING_ENABLED is false")
		<-ctx.Done()
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.signal:
			w.safePass(ctx)
		}
	}
}

// safePass runs a pass and recovers a panic, so tagging never takes the API
// server down with it (HLD section 3 risks).
func (w *Worker) safePass(ctx context.Context) {
	w.running.Store(true)
	defer w.running.Store(false)
	defer func() {
		if p := recover(); p != nil {
			w.cfg.Logger.Error("tagging pass panicked", "panic", fmt.Sprint(p))
		}
	}()
	rep, err := w.Pass(ctx)
	attrs := []any{"snapshot", rep.Snapshot, "tagged", rep.Tagged, "unresolved", rep.Unresolved,
		"calls", rep.Calls, "prompt_version", w.cfg.Prompt.Number}
	switch {
	case err == nil:
		w.cfg.Logger.Info("tagging pass done", attrs...)
	case ctx.Err() != nil:
		w.cfg.Logger.Info("tagging pass stopped by shutdown", attrs...)
	case isModelStop(err):
		w.cfg.Logger.Warn("tagging pass ended; the next import or restart retries", append(attrs, "err", err)...)
	default:
		w.cfg.Logger.Error("tagging pass failed", append(attrs, "err", err)...)
	}
}

func isModelStop(err error) bool {
	for _, target := range []error{gateway.ErrBudgetExhausted, gateway.ErrProviderCreditExhausted,
		gateway.ErrModelUnavailable, gateway.ErrRecordingMissing, gateway.ErrRecordingMismatch} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// Report counts what one pass did.
type Report struct {
	Snapshot   int // untagged reviews taken at the start
	Tagged     int // results stored
	Unresolved int // ids left for the next pass
	Calls      int // model requests
}

func (r *Report) add(o Report) {
	r.Snapshot += o.Snapshot
	r.Tagged += o.Tagged
	r.Unresolved += o.Unresolved
	r.Calls += o.Calls
}

// Pass tags a snapshot of the untagged reviews once, then the reviews that
// arrived during it, and ends; ids still unresolved wait for the next
// signal, so a review that never validates is not looped on (HLD section 16).
// Any model or store error ends the pass (HLD section 6).
func (w *Worker) Pass(ctx context.Context) (Report, error) {
	var total Report
	after := int64(0)
	for {
		rep, last, err := w.passFrom(ctx, after)
		total.add(rep)
		if err != nil || last == 0 {
			return total, err
		}
		newer, err := w.cfg.Store.UntaggedReviewIDs(ctx, last)
		if err != nil {
			return total, fmt.Errorf("check for newer reviews: %w", err)
		}
		if len(newer) == 0 {
			return total, nil
		}
		after = last
	}
}

// passFrom holds the lock over one snapshot of untagged ids above after and
// returns the snapshot's last id, 0 when it was empty.
func (w *Worker) passFrom(ctx context.Context, after int64) (Report, int64, error) {
	unlock, err := w.cfg.Store.LockTagging(ctx)
	if err != nil {
		return Report{}, 0, fmt.Errorf("take the tagging lock: %w", err)
	}
	defer unlock()

	ids, err := w.cfg.Store.UntaggedReviewIDs(ctx, after)
	if err != nil {
		return Report{}, 0, fmt.Errorf("snapshot untagged reviews: %w", err)
	}
	rep := Report{Snapshot: len(ids)}
	for start := 0; start < len(ids); start += BatchSize {
		batch := ids[start:min(start+BatchSize, len(ids))]
		b, err := w.tagBatch(ctx, batch)
		rep.add(b)
		if err != nil {
			rep.Unresolved += len(ids) - start - len(batch)
			return rep, 0, err
		}
	}
	if len(ids) == 0 {
		return rep, 0, nil
	}
	return rep, ids[len(ids)-1], nil
}

// tagBatch sends one batch, then at most MaxRetries more rounds for the ids
// that came back missing or invalid (one request) or repeated (one request
// each). Results are stored by review id after every call.
func (w *Worker) tagBatch(ctx context.Context, batch []int64) (Report, error) {
	reviews, err := w.cfg.Store.ReviewTexts(ctx, batch)
	if err != nil {
		return Report{}, fmt.Errorf("read review texts: %w", err)
	}
	texts := make(map[int64]string, len(reviews))
	pending := make([]int64, 0, len(reviews))
	for _, r := range reviews {
		texts[r.ID] = r.Text
		pending = append(pending, r.ID)
	}

	var rep Report
	var alone []int64
	for round := 0; round <= MaxRetries && len(pending)+len(alone) > 0; round++ {
		groups := make([][]int64, 0, 1+len(alone))
		if len(pending) > 0 {
			groups = append(groups, pending)
		}
		for _, id := range alone {
			groups = append(groups, []int64{id})
		}
		pending, alone = nil, nil
		for _, g := range groups {
			ans, err := w.call(ctx, g, texts)
			rep.Calls++
			if err != nil {
				rep.Unresolved = len(batch) - rep.Tagged
				return rep, err
			}
			n, err := w.cfg.Store.SaveResults(ctx, ans.Valid, w.cfg.Prompt.Number)
			if err != nil {
				rep.Unresolved = len(batch) - rep.Tagged
				return rep, fmt.Errorf("store tag results: %w", err)
			}
			rep.Tagged += n
			pending = append(pending, ans.Missing...)
			alone = append(alone, ans.Repeated...)
		}
		slices.Sort(pending)
		slices.Sort(alone)
	}
	rep.Unresolved = len(pending) + len(alone)
	w.cfg.Logger.Info("tagging batch", "first_id", batch[0], "size", len(batch), "calls", rep.Calls,
		"tagged", rep.Tagged, "unresolved", rep.Unresolved)
	return rep, nil
}

// call sends one request for ids and checks the answer against them.
func (w *Worker) call(ctx context.Context, ids []int64, texts map[int64]string) (Answer, error) {
	reviews := make([]Review, len(ids))
	for i, id := range ids {
		reviews[i] = Review{ID: id, Text: texts[id]}
	}
	resp, err := w.cfg.Model.Complete(ctx, gateway.Request{
		Purpose: gateway.Tagging,
		Prompt:  w.cfg.Prompt,
		User:    Message(reviews),
	})
	if err != nil {
		return Answer{}, err
	}
	ans := ParseAnswer(resp.Text, resp.FinishReason == "length", ids, Themes)
	if ans.Foreign > 0 {
		w.cfg.Logger.Warn("tagging answer named ids outside its batch; discarded", "lines", ans.Foreign)
	}
	return ans, nil
}
