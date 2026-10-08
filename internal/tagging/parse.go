package tagging

import (
	"slices"
	"strconv"
	"strings"
)

// Result is one review's validated tags, ready to store.
type Result struct {
	ReviewID int64
	// Themes are codes from the list, in list order; possibly none.
	Themes []string
	// Sentiment is positive, neutral or negative.
	Sentiment string
	// UrgentReasons are in the order of UrgentReasons; empty when not urgent.
	UrgentReasons []string
}

// IsUrgent is true exactly when a reason is present (data model, review_tags).
func (r Result) IsUrgent() bool { return len(r.UrgentReasons) > 0 }

// Answer is a model answer checked against its batch.
type Answer struct {
	// Valid holds one result per id that appeared once with a valid line.
	Valid []Result
	// Missing are batch ids with no line, or whose one line broke a rule:
	// both are retried together (AC-US-01-004-3, Q-014).
	Missing []int64
	// Repeated are batch ids on two or more lines: every one of those lines
	// is discarded and the id is retried alone, so one review's text cannot
	// write another's result (HLD section 3).
	Repeated []int64
	// Foreign counts lines naming an id outside the batch, discarded
	// (AC-US-01-004-2).
	Foreign int
}

var sentiments = map[string]string{"pos": "positive", "neu": "neutral", "neg": "negative"}

// ParseAnswer validates a tagging answer line by line, by review id, never by
// position (REQ-012, REQ-013, tenet 4). batch holds the ids sent, in order;
// themes is the theme list the prompt named. When the answer stopped at
// max_tokens, cut is true and the text after the last newline is dropped:
// a cut line can look valid with a reason missing.
func ParseAnswer(text string, cut bool, batch []int64, themes []Theme) Answer {
	if cut {
		if i := strings.LastIndexByte(text, '\n'); i >= 0 {
			text = text[:i]
		} else {
			text = ""
		}
	}
	inBatch := make(map[int64]bool, len(batch))
	for _, id := range batch {
		inBatch[id] = true
	}

	var a Answer
	seen := map[int64]int{}
	parsed := map[int64]Result{}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Split(strings.TrimSpace(line), "|")
		id, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
		if err != nil {
			continue // not a result line: a preamble, a blank line
		}
		if !inBatch[id] {
			a.Foreign++
			continue
		}
		seen[id]++
		if r, ok := parseFields(id, fields, themes); ok {
			parsed[id] = r
		}
	}

	for _, id := range batch {
		switch r, ok := parsed[id]; {
		case seen[id] > 1:
			a.Repeated = append(a.Repeated, id)
		case ok:
			a.Valid = append(a.Valid, r)
		default:
			a.Missing = append(a.Missing, id)
		}
	}
	return a
}

// parseFields checks the four fields of one line: id, themes, sentiment and
// urgent reasons. Codes are compared in lower case.
func parseFields(id int64, fields []string, themes []Theme) (Result, bool) {
	if len(fields) != 4 {
		return Result{}, false
	}
	themeCodes := make([]string, len(themes))
	for i, t := range themes {
		themeCodes[i] = t.Code
	}
	ts, ok := codeList(fields[1], themeCodes)
	if !ok {
		return Result{}, false
	}
	sentiment, ok := sentiments[strings.ToLower(strings.TrimSpace(fields[2]))]
	if !ok {
		return Result{}, false
	}
	reasons, ok := codeList(fields[3], UrgentReasons)
	if !ok {
		return Result{}, false
	}
	return Result{ReviewID: id, Themes: ts, Sentiment: sentiment, UrgentReasons: reasons}, true
}

// codeList reads "-" as none, or comma-separated codes from allowed with
// none repeated, and returns them in the order of allowed.
func codeList(field string, allowed []string) ([]string, bool) {
	field = strings.ToLower(strings.TrimSpace(field))
	if field == "-" {
		return []string{}, true
	}
	got := map[string]bool{}
	for _, c := range strings.Split(field, ",") {
		c = strings.TrimSpace(c)
		if !slices.Contains(allowed, c) || got[c] {
			return nil, false
		}
		got[c] = true
	}
	out := make([]string, 0, len(got))
	for _, c := range allowed {
		if got[c] {
			out = append(out, c)
		}
	}
	return out, true
}
