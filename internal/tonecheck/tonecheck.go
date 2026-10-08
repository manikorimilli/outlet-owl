// Package tonecheck drafts 30 replies for a person to score against the
// tone rubric (US-02-005), and totals the scores once they are filled in.
// The output is Markdown, not a spreadsheet, so no cell can carry a formula
// (HLD section 9). Report only: there is no pass mark (Q-019).
package tonecheck

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// Size is how many drafts the check holds (REQ-039).
const Size = 30

// Rubric is the brand tone of reply prompt v1 (GenAI design 4.2), one item
// per line, each scored 0 (missing), 1 (partly) or 2 (fully).
var Rubric = []string{
	"Thanks the reviewer, by first name when given",
	"Names the specific issue or praise",
	"Apologises without excuses when they complained",
	"Says what the outlet is doing, in general words",
	"Invites them back",
	"Signs as manager, outlet",
	"Promises no discount, refund or outcome; adds no facts",
	"Replies in the review's language and script",
}

func devanagari(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.Is(unicode.Devanagari, r) })
}

// Pick chooses 30 reviews spread evenly over the list, with at least 5 in
// Devanagari Hindi when the list holds them (AC-US-02-005-1).
func Pick(all []reviews.Review) []reviews.Review {
	if len(all) <= Size {
		return all
	}
	var hindi, others []reviews.Review
	for _, r := range all {
		if devanagari(r.Text) {
			hindi = append(hindi, r)
		} else {
			others = append(others, r)
		}
	}
	nh := min(5, len(hindi))
	out := spread(hindi, nh)
	out = append(out, spread(others, Size-nh)...)
	slices.SortFunc(out, func(a, b reviews.Review) int { return int(a.ID - b.ID) })
	return out
}

func spread(rs []reviews.Review, n int) []reviews.Review {
	out := make([]reviews.Review, 0, n)
	for i := range n {
		out = append(out, rs[i*len(rs)/n])
	}
	return out
}

// Draft is one review and the reply the model drafted for it.
type Draft struct {
	Review reviews.Review
	Text   string
}

// Write renders the scoring sheet.
func Write(w io.Writer, drafts []Draft, prompt prompts.Version, model string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Tone check\n\nReply prompt v%d, model %s, %d drafts. Score each rubric item 0 (missing), 1 (partly) or 2 (fully) by replacing the dash, then run `make tone-check TONE_ARGS=\"-report <this file>\"`.\n\n",
		prompt.Number, model, len(drafts))
	for i, d := range drafts {
		fmt.Fprintf(&b, "## Draft %d: review %d, %s, %d/5\n\nReview:\n\n%s\n\nDraft:\n\n%s\n\n| Item | Score |\n| --- | --- |\n",
			i+1, d.Review.ID, d.Review.OutletName, d.Review.Rating, quote(d.Review.Text), quote(d.Text))
		for _, item := range Rubric {
			fmt.Fprintf(&b, "| %s | - |\n", item)
		}
		b.WriteString("\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func quote(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

var (
	heading  = regexp.MustCompile(`^## Draft (\d+):`)
	scoreRow = regexp.MustCompile(`^\| (.+?) \| *([0-2-]) *\|$`)
)

// Totals reads a scored sheet and returns, per draft, the score of each
// rubric item (-1 when not scored yet).
func Totals(r io.Reader) ([][]int, error) {
	var out [][]int
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if heading.MatchString(line) {
			row := make([]int, len(Rubric))
			for i := range row {
				row[i] = -1
			}
			out = append(out, row)
			continue
		}
		m := scoreRow.FindStringSubmatch(line)
		if m == nil || len(out) == 0 {
			continue
		}
		item := slices.Index(Rubric, m[1])
		if item < 0 {
			continue
		}
		if m[2] != "-" {
			n, _ := strconv.Atoi(m[2])
			out[len(out)-1][item] = n
		}
	}
	return out, sc.Err()
}

// Summary renders the totals per draft and per rubric item
// (AC-US-02-005-2, -3), with no verdict.
func Summary(scores [][]int) string {
	var b strings.Builder
	unscored := 0
	b.WriteString("# Tone check scores (report only)\n\n| Draft | Total |\n| --- | --- |\n")
	perItem := make([]int, len(Rubric))
	for i, row := range scores {
		total := 0
		for j, s := range row {
			if s < 0 {
				unscored++
				continue
			}
			total += s
			perItem[j] += s
		}
		fmt.Fprintf(&b, "| %d | %d of %d |\n", i+1, total, 2*len(Rubric))
	}
	b.WriteString("\n| Rubric item | Total |\n| --- | --- |\n")
	for j, item := range Rubric {
		fmt.Fprintf(&b, "| %s | %d of %d |\n", item, perItem[j], 2*len(scores))
	}
	if unscored > 0 {
		fmt.Fprintf(&b, "\n%d scores are not filled in yet.\n", unscored)
	}
	return b.String()
}
