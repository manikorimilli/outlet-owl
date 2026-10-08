package tagging

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// memStore is the tagging store in memory, with the rule the SQL keeps: one
// result per review, never replaced.
type memStore struct {
	mu      sync.Mutex
	texts   map[int64]string
	tags    map[int64]Result
	version map[int64]int
	locks   int
}

func newMemStore(n int) *memStore {
	s := &memStore{texts: map[int64]string{}, tags: map[int64]Result{}, version: map[int64]int{}}
	for i := 1; i <= n; i++ {
		s.add(int64(i))
	}
	return s
}

func (s *memStore) add(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts[id] = fmt.Sprintf("review %d", id)
}

func (s *memStore) LockTagging(context.Context) (func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locks++
	return func() {}, nil
}

func (s *memStore) UntaggedReviewIDs(_ context.Context, after int64) ([]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []int64
	for id := range s.texts {
		if _, tagged := s.tags[id]; id > after && !tagged {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

func (s *memStore) ReviewTexts(_ context.Context, ids []int64) ([]Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Review
	for _, id := range ids {
		out = append(out, Review{ID: id, Text: s.texts[id]})
	}
	return out, nil
}

func (s *memStore) SaveResults(_ context.Context, rs []Result, v int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range rs {
		if _, ok := s.tags[r.ReviewID]; !ok {
			s.tags[r.ReviewID], s.version[r.ReviewID] = r, v
			n++
		}
	}
	return n, nil
}

// scriptModel answers each request through answer, which gets the ids in the
// request and the call number, and records every request's ids.
type scriptModel struct {
	mu     sync.Mutex
	calls  [][]int64
	answer func(call int, ids []int64) (string, error)
}

var reviewID = regexp.MustCompile(`<review id="(\d+)">`)

func (m *scriptModel) Complete(_ context.Context, r gateway.Request) (gateway.Response, error) {
	if r.Purpose != gateway.Tagging || r.Prompt.Name != "tagging" {
		return gateway.Response{}, fmt.Errorf("unexpected request %s %s", r.Purpose, r.Prompt.Name)
	}
	var ids []int64
	for _, m := range reviewID.FindAllStringSubmatch(r.User, -1) {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		ids = append(ids, id)
	}
	m.mu.Lock()
	m.calls = append(m.calls, ids)
	call := len(m.calls)
	m.mu.Unlock()
	text, err := m.answer(call, ids)
	return gateway.Response{Text: text, FinishReason: "stop"}, err
}

// allValid answers every id with one valid line, skipping those in skip.
func allValid(ids []int64, skip ...int64) string {
	var lines []string
	for _, id := range ids {
		if !slices.Contains(skip, id) {
			lines = append(lines, fmt.Sprintf("%d|food|neg|-", id))
		}
	}
	return strings.Join(lines, "\n")
}

func newWorker(s *memStore, m *scriptModel) *Worker {
	return New(Config{Store: s, Model: m, Prompt: prompts.Version{Name: "tagging", Number: 3, MaxTokens: 1000}, Enabled: true})
}

func sizes(calls [][]int64) []int {
	out := make([]int, len(calls))
	for i, c := range calls {
		out[i] = len(c)
	}
	return out
}

// AC-US-01-003-1, AC-US-02-002-2: 45 untagged reviews are three requests of
// 20, 20 and 5, and every stored result names the prompt version.
func TestPass_45ReviewsMakeThreeRequests(t *testing.T) {
	s := newMemStore(45)
	m := &scriptModel{answer: func(_ int, ids []int64) (string, error) { return allValid(ids), nil }}

	rep, err := newWorker(s, m).Pass(context.Background())

	if err != nil || rep.Tagged != 45 || !slices.Equal(sizes(m.calls), []int{20, 20, 5}) {
		t.Fatalf("report %+v, %v, request sizes %v; want 45 tagged in 20, 20, 5", rep, err, sizes(m.calls))
	}
	if s.version[1] != 3 || s.version[45] != 3 {
		t.Fatal("stored results must carry the prompt version")
	}
}

// AC-US-01-003-6: a second pass sends nothing for tagged reviews.
func TestPass_TaggedReviewsAreNotSentAgain(t *testing.T) {
	s := newMemStore(5)
	m := &scriptModel{answer: func(_ int, ids []int64) (string, error) { return allValid(ids), nil }}
	w := newWorker(s, m)
	if _, err := w.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Pass(context.Background()); err != nil || len(m.calls) != 1 {
		t.Fatalf("calls = %d, %v; want no second request", len(m.calls), err)
	}
}

// AC-US-01-004-4 and the retry that succeeds: 17 results are stored and the
// retry holds exactly the 3 missing ids.
func TestPass_RetriesOnlyMissingIDs(t *testing.T) {
	s := newMemStore(20)
	m := &scriptModel{answer: func(call int, ids []int64) (string, error) {
		if call == 1 {
			return allValid(ids, 4, 9, 17), nil
		}
		return allValid(ids), nil
	}}

	rep, err := newWorker(s, m).Pass(context.Background())

	if err != nil || len(m.calls) != 2 || !slices.Equal(m.calls[1], []int64{4, 9, 17}) || rep.Tagged != 20 || rep.Unresolved != 0 {
		t.Fatalf("report %+v, %v, calls %v; want the retry to hold 4, 9, 17", rep, err, m.calls)
	}
}

// AC-US-01-004-5 and the retry that gives up: an id never answered is sent
// three times in all, then left untagged for the next pass, which tries it
// again.
func TestPass_GivesUpAfterTwoRetries(t *testing.T) {
	s := newMemStore(20)
	m := &scriptModel{answer: func(_ int, ids []int64) (string, error) { return allValid(ids, 7), nil }}
	w := newWorker(s, m)

	rep, err := w.Pass(context.Background())

	if err != nil || len(m.calls) != 1+MaxRetries || rep.Tagged != 19 || rep.Unresolved != 1 {
		t.Fatalf("report %+v, %v, calls %v; want 3 calls, 19 tagged, 1 unresolved", rep, err, m.calls)
	}
	if _, tagged := s.tags[7]; tagged {
		t.Fatal("review 7 must stay untagged")
	}
	if _, err := w.Pass(context.Background()); err != nil || !slices.Equal(m.calls[len(m.calls)-1], []int64{7}) {
		t.Fatalf("the next pass must try 7 again; calls %v", m.calls)
	}
}

// HLD section 16: an id on two lines is retried in a request of its own.
func TestPass_DuplicateIDRetriedAlone(t *testing.T) {
	s := newMemStore(3)
	m := &scriptModel{answer: func(call int, ids []int64) (string, error) {
		if call == 1 {
			return "1|food|pos|-\n2|food|neg|harassment\n2|-|pos|-\n3|food|pos|-", nil
		}
		return allValid(ids), nil
	}}

	rep, err := newWorker(s, m).Pass(context.Background())

	if err != nil || len(m.calls) != 2 || !slices.Equal(m.calls[1], []int64{2}) || rep.Tagged != 3 {
		t.Fatalf("report %+v, %v, calls %v; want 2 retried alone", rep, err, m.calls)
	}
	if s.tags[2].IsUrgent() {
		t.Fatal("neither repeated line may be stored")
	}
}

// HLD section 6: a budget stop (or any gateway error) ends the pass.
func TestPass_GatewayErrorEndsPass(t *testing.T) {
	s := newMemStore(45)
	m := &scriptModel{answer: func(int, []int64) (string, error) { return "", gateway.ErrBudgetExhausted }}

	rep, err := newWorker(s, m).Pass(context.Background())

	if !isModelStop(err) || len(m.calls) != 1 || rep.Tagged != 0 || rep.Unresolved != 45 {
		t.Fatalf("report %+v, %v, %d calls; want the pass to end after one call", rep, err, len(m.calls))
	}
}

// HLD section 3: reviews imported during a pass are tagged after it, once.
func TestPass_RunsNewerReviewsAfterSnapshot(t *testing.T) {
	s := newMemStore(2)
	m := &scriptModel{answer: func(call int, ids []int64) (string, error) {
		if call == 1 {
			s.add(3)
		}
		return allValid(ids), nil
	}}

	rep, err := newWorker(s, m).Pass(context.Background())

	if err != nil || rep.Tagged != 3 || len(m.calls) != 2 || !slices.Equal(m.calls[1], []int64{3}) || s.locks != 2 {
		t.Fatalf("report %+v, %v, calls %v, locks %d; want 3 tagged in a second locked pass", rep, err, m.calls, s.locks)
	}
}

func TestWorker_PausedTagsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newMemStore(5)
		m := &scriptModel{answer: func(_ int, ids []int64) (string, error) { return allValid(ids), nil }}
		w := New(Config{Store: s, Model: m, Prompt: prompts.Version{Name: "tagging", Number: 1}, Enabled: false})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { w.Run(ctx); close(done) }()
		w.Signal()
		synctest.Wait() // Run is blocked for good: it would have tagged by now
		if len(m.calls) != 0 || s.locks != 0 {
			t.Fatalf("paused worker made %d calls", len(m.calls))
		}
		cancel()
		<-done
	})
}

func TestWorker_SignalsCoalesceAndRun(t *testing.T) {
	s := newMemStore(5)
	tagged := make(chan struct{}, 10)
	m := &scriptModel{answer: func(_ int, ids []int64) (string, error) {
		tagged <- struct{}{}
		return allValid(ids), nil
	}}
	w := newWorker(s, m)
	w.Signal()
	w.Signal()
	w.Signal()
	if len(w.signal) != 1 {
		t.Fatalf("pending signals = %d, want 1", len(w.signal))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)
	select {
	case <-tagged:
	case <-time.After(5 * time.Second):
		t.Fatal("a signal did not start a pass")
	}
}

// A panic inside a pass is recovered and the worker keeps serving signals.
func TestWorker_RecoversAPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newMemStore(1)
		m := &scriptModel{answer: func(call int, ids []int64) (string, error) {
			if call == 1 {
				panic("boom")
			}
			return allValid(ids), nil
		}}
		w := newWorker(s, m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go w.Run(ctx)
		w.Signal()
		synctest.Wait() // the first pass panicked and Run waits again
		w.Signal()
		synctest.Wait()
		if len(m.calls) != 2 || len(s.tags) != 1 {
			t.Fatalf("calls %d, tagged %d; want the second signal to tag after the panic", len(m.calls), len(s.tags))
		}
	})
}
