package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecord_SavesTheResponseUnderTheRequestKey(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	recorder, _ := liveGateway(t, Record, o, &fakeStore{})
	if _, err := recorder.Complete(context.Background(), request()); err != nil {
		t.Fatalf("record: %v", err)
	}
	store := &fakeStore{}
	player, err := New(Config{Mode: Replay, RecordingsDir: recorder.cfg.RecordingsDir, Store: store})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := player.Complete(context.Background(), request())

	if err != nil || resp.Text != "Thank you, Asha." || resp.Mode != Replay {
		t.Fatalf("replay = %+v, %v; want the recorded answer", resp, err)
	}
	if len(store.rows) != 0 || o.count() != 1 {
		t.Fatalf("replay wrote %d budget rows and the server saw %d requests; want 0 and 1 (AC-US-02-003-1)", len(store.rows), o.count())
	}
}

func TestReplay_MissingRecordingNamesItsKeyAndPath(t *testing.T) {
	g, _ := New(Config{Mode: Replay, RecordingsDir: t.TempDir()})

	_, err := g.Complete(context.Background(), request())

	var missing *RecordingMissingError
	if !errors.As(err, &missing) || !errors.Is(err, ErrRecordingMissing) {
		t.Fatalf("err = %v, want a RecordingMissingError (AC-US-02-003-2)", err)
	}
	body, _, _ := buildBody(request())
	if missing.Key != recordingKey(body) || !strings.Contains(err.Error(), missing.Path) || !strings.Contains(missing.Path, "drafting") {
		t.Fatalf("err = %v, want the key and the path under drafting/", err)
	}
}

func TestReplay_NeverBuildsAnHTTPClient(t *testing.T) {
	g, err := New(Config{Mode: Replay, APIKey: "would-be-ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if g.sender != nil {
		t.Fatal("replay mode built an HTTP sender; tests and CI must never reach the network (tenet 5)")
	}
	if r := g.Reconcile(context.Background()); r.State != "skipped" {
		t.Fatalf("Reconcile in replay = %+v, want skipped", r)
	}
}

func TestReplay_RefusesARecordingWhoseRequestDiffers(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: okBody})
	recorder, _ := liveGateway(t, Record, o, &fakeStore{})
	if _, err := recorder.Complete(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(recorder.cfg.RecordingsDir, "drafting", "*.json"))
	data, _ := os.ReadFile(files[0])
	if err := os.WriteFile(files[0], []byte(strings.Replace(string(data), "biryani", "dosa", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	player, _ := New(Config{Mode: Replay, RecordingsDir: recorder.cfg.RecordingsDir})

	_, err := player.Complete(context.Background(), request())

	if !errors.Is(err, ErrRecordingMismatch) {
		t.Fatalf("err = %v, want ErrRecordingMismatch", err)
	}
}

func TestRecord_FailedCallWritesNoFile(t *testing.T) {
	o := newOpenRouter(t, answer{status: 402})
	recorder, _ := liveGateway(t, Record, o, &fakeStore{})

	if _, err := recorder.Complete(context.Background(), request()); err == nil {
		t.Fatal("want the 402")
	}
	if files, _ := filepath.Glob(filepath.Join(recorder.cfg.RecordingsDir, "*", "*")); len(files) != 0 {
		t.Fatalf("a failed call wrote %v", files)
	}
}

func TestRecordingKey_ChangesWithThePromptVersion(t *testing.T) {
	a, _, _ := buildBody(request())
	r := request()
	r.Prompt.Text = "Reply in the brand's tone. Keep it short."
	b, _, _ := buildBody(r)

	if recordingKey(a) == recordingKey(b) || recordingKey(a) != recordingKey(append([]byte{}, a...)) {
		t.Fatal("the key must follow the exact request body")
	}
}

func TestReconcile_ProviderHigherInsertsTheDifference(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: `{"data":{"usage":0.5,"limit":10}}`})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	r := g.Reconcile(context.Background())

	if r.State != "added" || r.AddedUSD != "0.50000000" || len(store.providers) != 1 || store.providers[0] != "0.5" {
		t.Fatalf("Reconcile = %+v, store saw %v; want the provider's 0.5 recorded", r, store.providers)
	}
}

func TestReconcile_LocalHigherInsertsNothing(t *testing.T) {
	o := newOpenRouter(t, answer{status: 200, body: `{"data":{"usage":0.1}}`})
	g, _ := liveGateway(t, Live, o, &fakeStore{})

	if r := g.Reconcile(context.Background()); r.State != "not needed" {
		t.Fatalf("Reconcile = %+v, want not needed", r)
	}
}

func TestReconcile_KeyEndpointDownIsUnreconciled(t *testing.T) {
	o := newOpenRouter(t, answer{status: 500})
	store := &fakeStore{}
	g, _ := liveGateway(t, Live, o, store)

	r := g.Reconcile(context.Background())

	if r.State != "unreconciled" || r.Err == nil || len(store.providers) != 0 {
		t.Fatalf("Reconcile = %+v, want unreconciled with nothing written", r)
	}
}
