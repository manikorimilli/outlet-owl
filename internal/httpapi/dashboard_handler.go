package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
)

// DashboardService builds the reports. *dashboard.Service satisfies it.
type DashboardService interface {
	Trends(ctx context.Context, user auth.User) (dashboard.Trends, error)
	Heatmap(ctx context.Context, user auth.User) (dashboard.Heatmap, error)
	Movers(ctx context.Context, user auth.User) (dashboard.Movers, error)
}

func date(t time.Time) string { return t.Format(time.DateOnly) }

func ref(o outlets.Outlet) outletRef { return outletRef{ID: o.ID, Name: o.Name} }

func weekJSON(w dashboard.Week) map[string]string {
	return map[string]string{"start": date(w.Start), "end": date(w.End)}
}

// getTrends answers GET /api/v1/dashboard/trends (operationId getTrends).
func getTrends(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		t, err := d.Dashboard.Trends(r.Context(), user)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		type trendWeek struct {
			WeekStart     string   `json:"week_start"`
			ReviewCount   int      `json:"review_count"`
			AverageRating *float64 `json:"average_rating"`
			Positive      int      `json:"positive"`
			Neutral       int      `json:"neutral"`
			Negative      int      `json:"negative"`
			Untagged      int      `json:"untagged"`
			Replied       int      `json:"replied"`
		}
		type outletTrend struct {
			Outlet outletRef   `json:"outlet"`
			Weeks  []trendWeek `json:"weeks"`
		}
		weeks := make([]string, len(t.Weeks))
		for i, wk := range t.Weeks {
			weeks[i] = date(wk)
		}
		out := make([]outletTrend, len(t.Outlets))
		for i, o := range t.Outlets {
			ws := make([]trendWeek, len(o.Weeks))
			for j, x := range o.Weeks {
				ws[j] = trendWeek{WeekStart: date(x.WeekStart), ReviewCount: x.ReviewCount, AverageRating: x.AvgRating,
					Positive: x.Positive, Neutral: x.Neutral, Negative: x.Negative, Untagged: x.Untagged, Replied: x.Replied}
			}
			out[i] = outletTrend{Outlet: ref(o.Outlet), Weeks: ws}
		}
		WriteJSON(w, http.StatusOK, map[string]any{"weeks": weeks, "outlets": out})
	})
}

// getHeatmap answers GET /api/v1/dashboard/heatmap (operationId getHeatmap).
func getHeatmap(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		h, err := d.Dashboard.Heatmap(r.Context(), user)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		type row struct {
			Outlet outletRef      `json:"outlet"`
			Counts map[string]int `json:"counts"`
			Total  int            `json:"total"`
		}
		rows := make([]row, len(h.Rows))
		for i, x := range h.Rows {
			rows[i] = row{Outlet: ref(x.Outlet), Counts: x.Counts, Total: x.Total}
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"period": weekJSON(h.Period), "themes": themesJSON(), "rows": rows, "untagged_count": h.Untagged,
		})
	})
}

// getMovers answers GET /api/v1/dashboard/movers (operationId getMovers).
func getMovers(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		m, err := d.Dashboard.Movers(r.Context(), user)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		type mover struct {
			Outlet   outletRef `json:"outlet"`
			Theme    themeJSON `json:"theme"`
			Previous int       `json:"previous_count"`
			Current  int       `json:"current_count"`
			Change   int       `json:"change"`
		}
		movers := make([]mover, len(m.Movers))
		for i, x := range m.Movers {
			movers[i] = mover{Outlet: ref(x.Outlet), Theme: themeJSON{Code: x.Theme.Code, Label: x.Theme.Label},
				Previous: x.Previous, Current: x.Current, Change: x.Change}
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"week": weekJSON(m.Week), "previous_week": weekJSON(m.Previous),
			"review_count": m.ReviewCount, "untagged_count": m.Untagged, "movers": movers,
		})
	})
}
