// Package digest builds the weekly digest and sends it once to the brand
// admin through the local mail catcher (US-01-009, HLD flow C). It is
// generated on demand only (Q-005).
package digest

import (
	"fmt"
	"strings"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
)

var reasonLabels = map[string]string{"food_safety": "food safety", "harassment": "harassment", "legal_threat": "legal threat"}

func day(t time.Time) string { return t.Format("2 Jan 2006") }

func weekText(w dashboard.Week) string {
	return fmt.Sprintf("%s to %s", w.Start.Format("2 Jan"), day(w.End))
}

func change(n int) string {
	if n > 0 {
		return fmt.Sprintf("up %d", n)
	}
	return fmt.Sprintf("down %d", -n)
}

// Compose writes the subject and the plain-text body. The first line says
// how many reviews in the two weeks are untagged when any are (HLD flow C);
// the next names the outlet and theme that moved most with both counts
// (AC-US-01-009-3), then the ranked movers (-2) and the week's urgent
// reviews with their reasons (-4). Plain text only, so review text can never
// become markup (tenet 6).
func Compose(m dashboard.Movers, urgent []reviews.Review) (subject, body string) {
	var b strings.Builder
	if m.Untagged > 0 {
		fmt.Fprintf(&b, "%d reviews in these weeks are not tagged yet; movers and urgent reviews may be incomplete.\n", m.Untagged)
	}
	subject = fmt.Sprintf("Weekly digest, %s: no outlet and theme moved", weekText(m.Week))
	if len(m.Movers) > 0 {
		top := m.Movers[0]
		subject = fmt.Sprintf("Weekly digest, %s: %s %s %s", weekText(m.Week), top.Outlet.Name, strings.ToLower(top.Theme.Label), change(top.Change))
		fmt.Fprintf(&b, "Biggest mover: %s, %s. Negative reviews %d last week, %d this week (%s).\n",
			top.Outlet.Name, strings.ToLower(top.Theme.Label), top.Previous, top.Current, change(top.Change))
	} else {
		b.WriteString("Biggest mover: none. No outlet and theme changed in negative reviews.\n")
	}
	fmt.Fprintf(&b, "\nWeek: %s, compared with %s. %d reviews this week.\n", weekText(m.Week), weekText(m.Previous), m.ReviewCount)

	b.WriteString("\nMovers, by change in negative reviews\n")
	if len(m.Movers) == 0 {
		b.WriteString("None.\n")
	}
	for i, x := range m.Movers {
		if i == 10 {
			fmt.Fprintf(&b, "and %d more.\n", len(m.Movers)-10)
			break
		}
		fmt.Fprintf(&b, "%d. %s, %s: %d to %d (%s)\n", i+1, x.Outlet.Name, strings.ToLower(x.Theme.Label), x.Previous, x.Current, change(x.Change))
	}

	fmt.Fprintf(&b, "\nUrgent reviews this week: %d\n", len(urgent))
	for _, r := range urgent {
		var reasons []string
		if r.Tags != nil {
			for _, code := range r.Tags.UrgentReasons {
				reasons = append(reasons, reasonLabels[code])
			}
		}
		fmt.Fprintf(&b, "\n%s, %s, %d/5. Urgent: %s.\n%s\n", r.OutletName, day(r.ReviewDate), r.Rating, strings.Join(reasons, ", "),
			strings.TrimSpace(r.Text))
	}
	b.WriteString("\nSent by OutletOwl on request. Replies are drafted and approved in OutletOwl, then posted by hand.\n")
	return oneLine(subject), b.String()
}

// oneLine keeps a header value on one line, so an outlet name can never add
// a header.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
