package tonecheck

import (
	"bytes"
	"strings"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// AC-US-02-005-1: 30 reviews, Hindi among them, each draft with its prompt
// version on the sheet.
func TestPickAndWrite(t *testing.T) {
	var all []reviews.Review
	for i := range 200 {
		text := "Good food"
		if i%20 == 0 {
			text = "खाना अच्छा था"
		}
		all = append(all, reviews.Review{ID: int64(i + 1), Text: text, OutletName: "Koramangala", Rating: 4})
	}
	picked := Pick(all)
	hindi := 0
	for _, r := range picked {
		if devanagari(r.Text) {
			hindi++
		}
	}
	if len(picked) != Size || hindi != 5 {
		t.Fatalf("picked %d with %d in Hindi", len(picked), hindi)
	}
	var b bytes.Buffer
	drafts := []Draft{{Review: picked[0], Text: "Thank you!\nSee you soon."}}
	if err := Write(&b, drafts, prompts.Version{Number: 4}, "m"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "Reply prompt v4") || !strings.Contains(b.String(), "> Thank you!\n> See you soon.") {
		t.Fatalf("sheet:\n%s", b.String())
	}
}

// AC-US-02-005-2, -3: totals per draft and per rubric item, no verdict.
func TestTotalsAndSummary(t *testing.T) {
	var b bytes.Buffer
	_ = Write(&b, []Draft{{Text: "a"}, {Text: "b"}}, prompts.Version{Number: 1}, "m")
	sheet := strings.Replace(b.String(), "| "+Rubric[0]+" | - |", "| "+Rubric[0]+" | 2 |", 1)
	sheet = strings.Replace(sheet, "| "+Rubric[1]+" | - |", "| "+Rubric[1]+" | 1 |", 1)
	scores, err := Totals(strings.NewReader(sheet))
	if err != nil || len(scores) != 2 || scores[0][0] != 2 || scores[0][1] != 1 || scores[1][0] != -1 {
		t.Fatalf("scores = %v, %v", scores, err)
	}
	s := Summary(scores)
	if !strings.Contains(s, "| 1 | 3 of 16 |") || !strings.Contains(s, "14 scores are not filled in yet") || strings.Contains(strings.ToLower(s), "pass") {
		t.Fatalf("summary:\n%s", s)
	}
}
