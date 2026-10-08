// Package eval measures tagging on the 100 reviews the product owner labels
// (US-02-004): theme precision and recall, urgent recall and precision with
// the missed urgent reviews, and sentiment accuracy. Urgent recall below 90%
// fails the run (Q-017); the rest is report only (Q-018). It uses the same
// gateway, prompt, parser and retries as the tagging worker and never writes
// tag results.
package eval

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// GateRecall is the urgent recall the evaluation must reach (Q-017).
const GateRecall = 0.90

// Labelled is one labelled review, one JSON object per line of the set.
type Labelled struct {
	ID            int64    `json:"id"`
	Text          string   `json:"text"`
	Themes        []string `json:"themes"`
	Sentiment     string   `json:"sentiment"`
	UrgentReasons []string `json:"urgent_reasons"`
}

// Load reads the labelled set and checks every line against the theme list,
// the sentiments and the urgent reasons.
func Load(path string) ([]Labelled, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []Labelled
	seen := map[int64]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for line := 1; sc.Scan(); line++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var l Labelled
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		if err := check(l); err != nil || seen[l.ID] {
			return nil, fmt.Errorf("%s line %d: %v (or a repeated id)", path, line, err)
		}
		seen[l.ID] = true
		out = append(out, l)
	}
	return out, sc.Err()
}

func check(l Labelled) error {
	if l.ID < 1 || strings.TrimSpace(l.Text) == "" {
		return fmt.Errorf("id %d needs a positive id and text", l.ID)
	}
	for _, t := range l.Themes {
		if !slices.ContainsFunc(tagging.Themes, func(x tagging.Theme) bool { return x.Code == t }) {
			return fmt.Errorf("id %d: unknown theme %q", l.ID, t)
		}
	}
	if !slices.Contains([]string{"positive", "neutral", "negative"}, l.Sentiment) {
		return fmt.Errorf("id %d: sentiment %q", l.ID, l.Sentiment)
	}
	for _, r := range l.UrgentReasons {
		if !slices.Contains(tagging.UrgentReasons, r) {
			return fmt.Errorf("id %d: unknown urgent reason %q", l.ID, r)
		}
	}
	return nil
}

// Run tags the set in batches of 20 with the given purpose and returns each
// review's result; unresolved reviews are absent and count as tagged with
// nothing.
func Run(ctx context.Context, model tagging.Model, prompt prompts.Version, set []Labelled) (map[int64]tagging.Result, int, error) {
	got := map[int64]tagging.Result{}
	calls := 0
	for start := 0; start < len(set); start += tagging.BatchSize {
		var batch []tagging.Review
		for _, l := range set[start:min(start+tagging.BatchSize, len(set))] {
			batch = append(batch, tagging.Review{ID: l.ID, Text: l.Text})
		}
		_, n, err := tagging.Classify(ctx, model, prompt, gateway.Evaluation, batch, func(rs []tagging.Result) error {
			for _, r := range rs {
				got[r.ReviewID] = r
			}
			return nil
		}, nil)
		calls += n
		if err != nil {
			return got, calls, err
		}
	}
	return got, calls, nil
}

// Metrics is what the report shows.
type Metrics struct {
	Reviews, Unresolved                        int
	Theme                                      map[string][2]float64 // precision, recall
	UrgentRecall, UrgentPrecision              float64
	UrgentLabelled                             int
	Missed                                     []Labelled
	SentimentAccuracy, NegPrecision, NegRecall float64
	Passed                                     bool
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// Measure compares the results with the labels.
func Measure(set []Labelled, got map[int64]tagging.Result) Metrics {
	m := Metrics{Reviews: len(set), Theme: map[string][2]float64{}}
	type count struct{ tp, fp, fn int }
	themes := map[string]*count{}
	for _, t := range tagging.Themes {
		themes[t.Code] = &count{}
	}
	var urgent, neg count
	correctSentiment := 0
	for _, l := range set {
		r, ok := got[l.ID]
		if !ok {
			m.Unresolved++
		}
		for _, t := range tagging.Themes {
			want, have := slices.Contains(l.Themes, t.Code), slices.Contains(r.Themes, t.Code)
			c := themes[t.Code]
			switch {
			case want && have:
				c.tp++
			case have:
				c.fp++
			case want:
				c.fn++
			}
		}
		wantU, haveU := len(l.UrgentReasons) > 0, r.IsUrgent()
		switch {
		case wantU && haveU:
			urgent.tp++
		case haveU:
			urgent.fp++
		case wantU:
			urgent.fn++
			m.Missed = append(m.Missed, l)
		}
		if wantU {
			m.UrgentLabelled++
		}
		if ok && r.Sentiment == l.Sentiment {
			correctSentiment++
		}
		wantN, haveN := l.Sentiment == "negative", r.Sentiment == "negative"
		switch {
		case wantN && haveN:
			neg.tp++
		case haveN:
			neg.fp++
		case wantN:
			neg.fn++
		}
	}
	for code, c := range themes {
		m.Theme[code] = [2]float64{ratio(c.tp, c.tp+c.fp), ratio(c.tp, c.tp+c.fn)}
	}
	m.UrgentRecall, m.UrgentPrecision = ratio(urgent.tp, urgent.tp+urgent.fn), ratio(urgent.tp, urgent.tp+urgent.fp)
	m.SentimentAccuracy = ratio(correctSentiment, len(set))
	m.NegPrecision, m.NegRecall = ratio(neg.tp, neg.tp+neg.fp), ratio(neg.tp, neg.tp+neg.fn)
	m.Passed = m.UrgentLabelled > 0 && m.UrgentRecall >= GateRecall
	return m
}

func pct(f float64) string { return fmt.Sprintf("%.0f%%", f*100) }

// Report writes the Markdown report (AC-US-02-004-1 to -6).
func Report(m Metrics, prompt prompts.Version, model string, mode gateway.Mode, calls int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Tagging evaluation\n\nModel %s, tagging prompt v%d, gateway mode %s, %d calls. %d reviews, %d unresolved (counted as tagged with nothing).\n\n",
		model, prompt.Number, mode, calls, m.Reviews, m.Unresolved)
	verdict := "FAILED"
	if m.Passed {
		verdict = "PASSED"
	}
	fmt.Fprintf(&b, "## Urgent gate: %s\n\nUrgent recall %s (gate %s), urgent precision %s, %d labelled urgent.\n\n",
		verdict, pct(m.UrgentRecall), pct(GateRecall), pct(m.UrgentPrecision), m.UrgentLabelled)
	if len(m.Missed) > 0 {
		b.WriteString("Missed urgent reviews:\n\n")
		for _, l := range m.Missed {
			text := []rune(strings.Join(strings.Fields(l.Text), " "))
			if len(text) > 120 {
				text = append(text[:120], []rune("...")...)
			}
			fmt.Fprintf(&b, "- %d (%s): %s\n", l.ID, strings.Join(l.UrgentReasons, ", "), string(text))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Themes (report only)\n\n| Theme | Precision | Recall |\n| --- | --- | --- |\n")
	for _, t := range tagging.Themes {
		v := m.Theme[t.Code]
		fmt.Fprintf(&b, "| %s | %s | %s |\n", t.Label, pct(v[0]), pct(v[1]))
	}
	fmt.Fprintf(&b, "\n## Sentiment (report only)\n\nAccuracy %s. Negative precision %s, negative recall %s.\n",
		pct(m.SentimentAccuracy), pct(m.NegPrecision), pct(m.NegRecall))
	return b.String()
}
