package gateway

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeRow is one budget.model_calls row as the fake store keeps it.
type fakeRow struct {
	purpose, reserved, settled, outcome string
	promptVersion, in, out              int
}

// fakeStore applies the store's rules in memory: settle and fail change only
// a reserved row; refuse makes the next reserve fail as over the limit.
type fakeStore struct {
	mu        sync.Mutex
	rows      []fakeRow
	refuseAt  int // the reserve with this 1-based number and later are refused; 0 never
	providers []string
}

func (f *fakeStore) ReserveModelCall(_ context.Context, purpose, _ string, pv int, reserved, _ string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.refuseAt > 0 && len(f.rows)+1 >= f.refuseAt {
		return 0, ErrBudgetExhausted
	}
	f.rows = append(f.rows, fakeRow{purpose: purpose, reserved: reserved, outcome: "reserved", promptVersion: pv})
	return int64(len(f.rows)), nil
}

func (f *fakeStore) SettleModelCall(_ context.Context, id int64, in, out int, cost string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := &f.rows[id-1]
	if r.outcome != "reserved" {
		return false, nil
	}
	r.outcome, r.in, r.out, r.settled = "settled", in, out, cost
	return true, nil
}

func (f *fakeStore) FailModelCall(_ context.Context, id int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := &f.rows[id-1]
	if r.outcome != "reserved" {
		return false, nil
	}
	r.outcome = "failed"
	return true, nil
}

func (f *fakeStore) RunningTotalUSD(context.Context) (string, error) { return "0", nil }

func (f *fakeStore) InsertReconciliation(_ context.Context, usage string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers = append(f.providers, usage)
	if usage == "0.5" {
		return "0.50000000", true, nil
	}
	return "", false, nil
}

func (f *fakeStore) outcomes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.rows {
		out = append(out, r.outcome)
	}
	return out
}

// answer is one canned OpenRouter reply.
type answer struct {
	status int
	body   string
	delay  time.Duration
}

const okBody = `{"choices":[{"message":{"content":"Thank you, Asha."},"finish_reason":"stop"}],` +
	`"usage":{"prompt_tokens":120,"completion_tokens":8,"cost":0.00016}}`

// openRouter is a loopback server that answers like OpenRouter (decided in
// session: nothing leaves the machine, no key, no spend).
type openRouter struct {
	*httptest.Server
	mu       sync.Mutex
	answers  []answer
	requests [][]byte
}

func newOpenRouter(t *testing.T, answers ...answer) *openRouter {
	t.Helper()
	o := &openRouter{answers: answers}
	o.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		o.mu.Lock()
		o.requests = append(o.requests, body)
		n := len(o.requests)
		a := o.answers[min(n, len(o.answers))-1]
		o.mu.Unlock()
		if a.delay > 0 {
			select {
			case <-time.After(a.delay):
			case <-r.Context().Done():
				return
			}
		}
		w.WriteHeader(a.status)
		_, _ = w.Write([]byte(a.body))
	}))
	t.Cleanup(o.Close)
	return o
}

func (o *openRouter) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.requests)
}

// liveGateway returns a gateway in mode pointed at the loopback server, with
// millisecond backoff, and the buffer its log lines go to.
func liveGateway(t *testing.T, mode Mode, o *openRouter, store *fakeStore) (*Gateway, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	g, err := New(Config{
		Mode: mode, APIKey: "test-key", BaseURL: o.URL, Store: store,
		RecordingsDir:  t.TempDir(),
		Logger:         slog.New(slog.NewJSONHandler(&logs, nil)),
		AttemptTimeout: 2 * time.Second,
		Backoff:        []time.Duration{time.Millisecond, time.Millisecond},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g, &logs
}
