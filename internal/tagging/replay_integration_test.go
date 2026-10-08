//go:build integration

package tagging_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/imports"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// TestPassThroughReplayGateway runs a pass through the real gateway in
// replay mode against PostgreSQL (tenet 1, tenet 5, AC-US-02-002-2,
// AC-US-01-003-6). The recording is synthetic, written by this test: it
// proves the plumbing, not what the model would answer.
func TestPassThroughReplayGateway(t *testing.T) {
	db := storetest.New(t)
	st := &store.Store{Pool: db.Pool}
	ctx := context.Background()
	o, _, err := st.CreateOutlet(ctx, "Indiranagar")
	if err != nil {
		t.Fatal(err)
	}
	var reviews []imports.NewReview
	for i, text := range []string{"Food was cold and the waiter was rude", "Found a hair in my dal", "Lovely place"} {
		reviews = append(reviews, imports.NewReview{OutletID: o.ID, Row: connector.Row{Number: i + 2, Outlet: "Indiranagar",
			Source: "Google", Date: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), Rating: 3, Text: text, ReviewerName: "Asha"}})
	}
	if _, _, err := st.SaveImport(ctx, imports.Write{RequestID: "5192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "f.csv", Reviews: reviews}); err != nil {
		t.Fatal(err)
	}
	ids, _ := st.UntaggedReviewIDs(ctx, 0)

	dir := t.TempDir()
	gw, err := gateway.New(gateway.Config{Mode: gateway.Replay, RecordingsDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	reg, err := prompts.Load()
	if err != nil {
		t.Fatal(err)
	}
	prompt, _ := reg.Current("tagging")
	w := tagging.New(tagging.Config{Store: st, Model: gw, Prompt: prompt, Enabled: true})

	// No recording yet: replay refuses and the pass ends with nothing stored.
	_, err = w.Pass(ctx)
	var missing *gateway.RecordingMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("first pass err = %v, want a missing recording", err)
	}

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	body, _ := json.Marshal(struct {
		Model     string    `json:"model"`
		Messages  []message `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
		Temp      *float64  `json:"temperature,omitempty"`
	}{gateway.Model, []message{
		{Role: "system", Content: prompt.Text},
		{Role: "user", Content: tagging.Message([]tagging.Review{{ID: ids[0], Text: reviews[0].Text}, {ID: ids[1], Text: reviews[1].Text}, {ID: ids[2], Text: reviews[2].Text}})},
	}, prompt.MaxTokens, prompt.Temperature})
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != missing.Key {
		t.Fatalf("the test's request body differs from the gateway's: key %s, want %s", hex.EncodeToString(sum[:]), missing.Key)
	}
	answer := fmt.Sprintf("%d|food,staff|neg|-\n%d|food|neg|food_safety\n%d|-|pos|-", ids[0], ids[1], ids[2])
	resp, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"message": map[string]string{"content": answer}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": 900, "completion_tokens": 30, "cost": 0.00105},
	})
	rec, _ := json.Marshal(map[string]any{"key": missing.Key, "purpose": "tagging",
		"prompt":  map[string]any{"name": "tagging", "version": prompt.Number},
		"request": json.RawMessage(body), "response": json.RawMessage(resp)})
	if err := os.MkdirAll(filepath.Dir(missing.Path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(missing.Path, rec, 0o600); err != nil {
		t.Fatal(err)
	}

	rep, err := w.Pass(ctx)
	if err != nil || rep.Tagged != 3 || rep.Calls != 1 {
		t.Fatalf("second pass = %+v, %v; want 3 tagged in one call", rep, err)
	}
	var reasons []string
	var version int
	if err := db.Pool.QueryRow(ctx, "SELECT urgent_reasons, prompt_version FROM review_tags WHERE review_id = $1", ids[1]).
		Scan(&reasons, &version); err != nil || !slices.Equal(reasons, []string{"food_safety"}) || version != prompt.Number {
		t.Fatalf("stored %v v%d (%v); want food_safety with prompt v%d", reasons, version, err, prompt.Number)
	}

	// A third pass has nothing to send: tagged reviews never reach the model.
	if rep, err := w.Pass(ctx); err != nil || rep.Calls != 0 {
		t.Fatalf("third pass = %+v, %v; want no call", rep, err)
	}
	var rows int
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM budget.model_calls").Scan(&rows)
	if rows != 0 {
		t.Fatalf("replay wrote %d budget rows, want 0", rows)
	}
}
