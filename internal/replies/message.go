package replies

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxReviewRunes = 2000

// Input is what a reply draft is written from (GenAI design 4.2).
type Input struct {
	Outlet       string
	ManagerName  string
	ReviewerName string
	Rating       int
	Text         string
}

// FirstName is the first word of a name, or "" when that word is one
// letter or holds anything but letters and marks (a nickname, an initial,
// a handle): the draft then has no name greeting.
func FirstName(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	w := fields[0]
	if utf8.RuneCountInString(w) < 2 {
		return ""
	}
	for _, r := range w {
		if !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) && !unicode.Is(unicode.Mc, r) {
			return ""
		}
	}
	return w
}

// Message builds the reply call's user message. The review text is cut at
// 2,000 characters and cannot close its own tag.
func Message(in Input) string {
	text := strings.ReplaceAll(in.Text, "</review", "")
	if utf8.RuneCountInString(text) > maxReviewRunes {
		text = string([]rune(text)[:maxReviewRunes])
	}
	reviewer := FirstName(in.ReviewerName)
	if reviewer == "" {
		reviewer = "not given"
	}
	signer := FirstName(in.ManagerName)
	if signer == "" {
		signer = strings.TrimSpace(in.ManagerName)
	}
	return fmt.Sprintf("Outlet: %s\nSignature: %s, outlet manager, %s\nReviewer first name: %s\nRating: %d of 5\n<review>%s</review>",
		in.Outlet, signer, in.Outlet, reviewer, in.Rating, text)
}
