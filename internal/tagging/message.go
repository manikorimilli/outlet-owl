package tagging

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxTextRunes is where a review's text is cut before it is sent (HLD
// section 8).
const maxTextRunes = 2000

// Review is one review as the model receives it: its id and text only, no
// name, outlet or date (GenAI design 4.1), so the request is the same on
// every run and its recording replays.
type Review struct {
	ID   int64
	Text string
}

// Message builds the user message: one <review id="N">text</review> line
// per review, in the order given.
func Message(reviews []Review) string {
	var b strings.Builder
	for i, r := range reviews {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(`<review id="`)
		b.WriteString(strconv.FormatInt(r.ID, 10))
		b.WriteString(`">`)
		b.WriteString(cleanText(r.Text))
		b.WriteString("</review>")
	}
	return b.String()
}

// cleanText removes every "<review" and "</review", ignoring ASCII case,
// until none is left, so a review cannot open or close another; then cuts
// the text at 2,000 characters.
func cleanText(s string) string {
	for {
		i, n := indexASCIIFold(s, "</review"), len("</review")
		if j := indexASCIIFold(s, "<review"); j >= 0 && (i < 0 || j < i) {
			i, n = j, len("<review")
		}
		if i < 0 {
			break
		}
		s = s[:i] + s[i+n:]
	}
	if utf8.RuneCountInString(s) > maxTextRunes {
		s = string([]rune(s)[:maxTextRunes])
	}
	return s
}

// indexASCIIFold finds pat, which is lower-case ASCII, in s ignoring ASCII
// case. Byte offsets stay those of s, which strings.ToLower does not promise.
func indexASCIIFold(s, pat string) int {
	for i := 0; i+len(pat) <= len(s); i++ {
		match := true
		for k := 0; k < len(pat); k++ {
			c := s[i+k]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != pat[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
