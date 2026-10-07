package gateway

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestComplete_ReservesThenSettlesWithUsageCost(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	resp, err := g.Complete(context.Background(), request())

	if err != nil || resp.Text != "Thank you, Asha." || resp.Mode != Live {
		t.Fatalf("Complete = %+v, %v", resp, err)
	}
	r := store.rows[0]
	if r.outcome != "settled" || r.in != 120 || r.out != 8 || r.settled != "0.00016" || r.purpose != "drafting" || r.promptVersion != 3 {
		t.Fatalf("row = %+v, want settled with the usage tokens and cost (AC-US-02-001-2)", r)
	}
}

func TestComplete_KeepsTheReservedPriceWhenUsageCostIsMissing(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: `{"choices":[{"message":{"content":"x"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1}}`})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	if _, err := g.Complete(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if r := store.rows[0]; r.settled != r.reserved || r.reserved == "" {
		t.Fatalf("row = %+v, want the reserved price as the settled cost", r)
	}
}

func TestComplete_RefusedBudgetSendsNothing(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	g, _ := liveGateway(t, Live, o, &fakeStore{refuseAt: 1})

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrBudgetExhausted) || o.count() != 0 {
		t.Fatalf("err = %v, requests = %d; want ErrBudgetExhausted and nothing sent (AC-US-02-001-4)", err, o.count())
	}
}

func TestComplete_Retries429ThenSucceeds(t *testing.T) {
	o := newOpenRouter(t, answer{status: 429}, answer{status: 200, body: okBody})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	if _, err := g.Complete(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if got := store.outcomes(); !reflect.DeepEqual(got, []string{"failed", "settled"}) || o.count() != 2 {
		t.Fatalf("rows %v after %d requests, want failed then settled", got, o.count())
	}
}

func TestComplete_GivesUpAfterTwoRetries(t *testing.T) {
	o := newOpenRouter(t, answer{status: 503})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrModelUnavailable) || o.count() != 3 {
		t.Fatalf("err = %v after %d requests, want ErrModelUnavailable after 3", err, o.count())
	}
	if got := store.outcomes(); !reflect.DeepEqual(got, []string{"failed", "failed", "failed"}) {
		t.Fatalf("rows = %v, want three failed attempts, each counted", got)
	}
}

func TestComplete_RetryRechecksTheBudget(t *testing.T) {
	o := newOpenRouter(t, answer{status: 429}, answer{status: 200, body: okBody})
	g, _ := liveGateway(t, Live, o, &fakeStore{refuseAt: 2})

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrBudgetExhausted) || o.count() != 1 {
		t.Fatalf("err = %v after %d requests, want ErrBudgetExhausted after 1", err, o.count())
	}
}

func TestComplete_402StopsWithoutRetry(t *testing.T) {
	o := newOpenRouter(t, answer{status: 402, body: `{"error":{"code":402,"metadata":{"limit_source":"key"}}}`})
	g, _ := liveGateway(t, Live, o, &fakeStore{})

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrProviderCreditExhausted) || o.count() != 1 || !strings.Contains(err.Error(), "limit source key") {
		t.Fatalf("err = %v after %d requests, want ErrProviderCreditExhausted after 1", err, o.count())
	}
}

func TestComplete_MissingModelFailsWithNoFallback(t *testing.T) {
	o := newOpenRouter(t, answer{status: 404, body: `{"error":{"message":"No endpoints found"}}`})
	g, _ := liveGateway(t, Live, o, &fakeStore{})

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrModelUnavailable) || o.count() != 1 {
		t.Fatalf("err = %v after %d requests, want ErrModelUnavailable and no second model (AC-US-02-001-7)", err, o.count())
	}
}

func TestComplete_TimeoutFailsTheRowAtItsReservedPrice(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody, delay: time.Second})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)
	g.cfg.AttemptTimeout = 50 * time.Millisecond

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrModelUnavailable) || o.count() != 1 {
		t.Fatalf("err = %v after %d requests, want ErrModelUnavailable and no resend", err, o.count())
	}
	if got := store.outcomes(); !reflect.DeepEqual(got, []string{"failed"}) {
		t.Fatalf("rows = %v, want one failed row at its reserved price", got)
	}
}

func TestComplete_CancelledCallerStillFailsTheRow(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody, delay: time.Second})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := g.Complete(ctx, request()); err == nil {
		t.Fatal("want an error from the cancelled call")
	}
	if got := store.outcomes(); !reflect.DeepEqual(got, []string{"failed"}) {
		t.Fatalf("rows = %v, want the row failed, not left reserved", got)
	}
}

func TestComplete_UnreadableBodyIsModelUnavailable(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: `<html>busy</html>`})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, ErrModelUnavailable) || !reflect.DeepEqual(store.outcomes(), []string{"failed"}) {
		t.Fatalf("err = %v, rows %v; want ErrModelUnavailable and a failed row", err, store.outcomes())
	}
}

func TestComplete_LogLineCarriesNoPromptOrUserText(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	g, logs := liveGateway(t, Live, o, &fakeStore{})

	if _, err := g.Complete(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	out := logs.String()
	if !strings.Contains(out, `"msg":"model call"`) || !strings.Contains(out, `"settled_usd":"0.00016"`) {
		t.Fatalf("log = %s, want the model call line with its cost", out)
	}
	for _, secret := range []string{"biryani", testPrompt.Text, "Asha", "test-key"} {
		if strings.Contains(out, secret) {
			t.Fatalf("log carries %q: %s", secret, out)
		}
	}
}

func TestNew_LiveNeedsAKey(t *testing.T) {
	if _, err := New(Config{Mode: Live, Store: &fakeStore{}}); err == nil {
		t.Fatal("live mode without a key must be refused")
	}
}

// The cost row of a paid call is written even when the caller has already
// given up: settle and fail must not use the caller's context (LLD 4.1).
func TestSettleAndFail_IgnoreTheCallersCancellation(t *testing.T) {
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, newOpenRouter(t, answer{status: 200, body: okBody}), store)
	for i := 0; i < 2; i++ {
		if _, err := store.ReserveModelCall(context.Background(), "drafting", Model, 1, "0.01", LimitUSD); err != nil {
			t.Fatal(err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	g.settle(cancelled, 1, 10, 2, "0.0001")
	g.fail(cancelled, 2)

	if got := store.outcomes(); !reflect.DeepEqual(got, []string{"settled", "failed"}) {
		t.Fatalf("rows = %v after a cancelled caller, want settled and failed", got)
	}
}

func TestComplete_DatabaseFaultIsLoggedAsAnErrorNotARefusal(t *testing.T) {
	fault := errors.New("connection refused")
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	g, logs := liveGateway(t, Live, o, &fakeStore{reserveErr: fault})

	_, err := g.Complete(context.Background(), request())

	if !errors.Is(err, fault) || o.count() != 0 {
		t.Fatalf("err = %v after %d requests, want the fault and nothing sent", err, o.count())
	}
	if out := logs.String(); !strings.Contains(out, `"outcome":"error"`) || strings.Contains(out, `"outcome":"refused"`) {
		t.Fatalf("log = %s, want outcome error, not refused", out)
	}
}
