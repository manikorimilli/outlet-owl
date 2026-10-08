package eval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// answerModel answers with fixed lines; a synthetic fixture, not a model.
type answerModel struct{ text string }

func (a answerModel) Complete(context.Context, gateway.Request) (gateway.Response, error) {
	return gateway.Response{Text: a.text, FinishReason: "stop"}, nil
}

var v1 = prompts.Version{Name: "tagging", Number: 1, MaxTokens: 1000}

// AC-US-02-004-2, -4, -6: one of two urgent reviews missed is 50% recall,
// below the gate; the missed review is listed; sentiment is reported.
func TestRunAndMeasure_FailsTheGateAndListsMisses(t *testing.T) {
	set, err := Load("testdata/sample.jsonl")
	if err != nil || len(set) != 4 {
		t.Fatalf("load = %d, %v", len(set), err)
	}
	model := answerModel{text: "1|food,cleanliness|neg|food_safety\n2|price|neg|-\n3|food|pos|-\n4|wait_time|neu|-"}
	got, calls, err := Run(context.Background(), model, v1, set)
	if err != nil || calls != 1 || len(got) != 4 {
		t.Fatalf("run = %d results, %d calls, %v", len(got), calls, err)
	}
	m := Measure(set, got)
	if m.Passed || m.UrgentRecall != 0.5 || m.UrgentPrecision != 1 || len(m.Missed) != 1 || m.Missed[0].ID != 2 {
		t.Fatalf("metrics = %+v", m)
	}
	if m.SentimentAccuracy != 0.75 || m.NegRecall != 2.0/3.0 || m.Theme["food"] != [2]float64{1, 1} {
		t.Fatalf("sentiment and themes = %+v", m)
	}
	r := Report(m, v1, gateway.Replay, calls)
	for _, want := range []string{"Urgent gate: FAILED", "tagging prompt v1", gateway.Model, "- 2 (legal_threat): I will take you to consumer court"} {
		if !strings.Contains(r, want) {
			t.Errorf("report lacks %q:\n%s", want, r)
		}
	}
}

// AC-US-02-004-3: every urgent review found passes the gate.
func TestMeasure_PassesAtFullRecall(t *testing.T) {
	set, _ := Load("testdata/sample.jsonl")
	got, _, _ := Run(context.Background(), answerModel{text: "1|food|neg|food_safety\n2|price|neg|legal_threat\n3|food|pos|-\n4|wait_time|neg|-"}, v1, set)
	if m := Measure(set, got); !m.Passed || m.UrgentRecall != 1 {
		t.Fatalf("metrics = %+v", m)
	}
}

func TestLoad_RefusesBadLabels(t *testing.T) {
	dir := t.TempDir()
	for _, line := range []string{
		`{"id":1,"text":"x","themes":["ambience"],"sentiment":"negative","urgent_reasons":[]}`,
		`{"id":1,"text":"x","themes":[],"sentiment":"angry","urgent_reasons":[]}`,
		`{"id":1,"text":"x","themes":[],"sentiment":"negative","urgent_reasons":["fire"]}`,
		`{"id":0,"text":"x","themes":[],"sentiment":"negative","urgent_reasons":[]}`,
	} {
		p := filepath.Join(dir, "set.jsonl")
		_ = os.WriteFile(p, []byte(line+"\n"), 0o600)
		if _, err := Load(p); err == nil {
			t.Errorf("accepted %s", line)
		}
	}
}
