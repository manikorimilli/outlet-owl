// Package dashboard builds the weekly reports: trends, the theme heatmap and
// the movers (US-01-005 to US-01-007, phase 4 LLD sections 3 and 6). Weeks
// run Monday to Sunday in the brand timezone (Q-005); "now" comes from an
// injected clock so tests pin it.
package dashboard

import "time"

// Week is one Monday to Sunday week; dates are calendar dates at midnight UTC,
// as PostgreSQL date values arrive.
type Week struct {
	Start time.Time
	End   time.Time
}

func weekFrom(monday time.Time) Week { return Week{Start: monday, End: monday.AddDate(0, 0, 6)} }

// LatestCompleteWeek is the last Monday to Sunday week that ended before
// today in loc.
func LatestCompleteWeek(now time.Time, loc *time.Location) Week {
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	sinceMonday := (int(today.Weekday()) + 6) % 7
	thisMonday := today.AddDate(0, 0, -sinceMonday)
	return weekFrom(thisMonday.AddDate(0, 0, -7))
}

// lastWeeks returns the n weeks ending with latest, oldest first.
func lastWeeks(latest Week, n int) []Week {
	out := make([]Week, n)
	for i := range n {
		out[i] = weekFrom(latest.Start.AddDate(0, 0, -7*(n-1-i)))
	}
	return out
}
