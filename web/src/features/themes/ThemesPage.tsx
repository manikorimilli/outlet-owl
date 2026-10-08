import { Link } from "react-router";
import { useSession } from "../../app/session-context";
import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";
import { formatRange } from "../../lib/format";
import { useLoad } from "../../lib/use-load";
import { reviewsLink } from "../reviews/api";
import { TableScroll } from "../../components/TableScroll";

type Heatmap = components["schemas"]["Heatmap"];
const getHeatmap = () => apiFetch<Heatmap>("/dashboard/heatmap");

// The ramp's buckets, printed in the legend; the count is in every cell, so
// colour is never the only signal (AC-US-01-006-3).
const buckets = [
  { max: 0, label: "0" },
  { max: 3, label: "1 to 3" },
  { max: 6, label: "4 to 6" },
  { max: 9, label: "7 to 9" },
  { max: 14, label: "10 to 14" },
  { max: Infinity, label: "15 or more" },
];
const level = (n: number) => buckets.findIndex((b) => n <= b.max);

// ThemesPage is S-03: negative reviews by outlet and theme over 4 weeks.
export function ThemesPage() {
  const { session } = useSession();
  const me = session.status === "signed-in" ? session.me : null;
  const isAdmin = me?.role === "brand_admin";
  const [state, reload] = useLoad("heatmap", getHeatmap);
  const scope = isAdmin || !me?.outlet ? "All outlets" : me.outlet.name;
  const h = state.status === "ready" ? state.data : null;
  const empty = h !== null && h.rows.every((r) => r.total === 0);

  return (
    <div className="content">
      <div className="page-head">
        <h1>
          <span className="hl">Themes</span>
        </h1>
        <p className="muted">
          {scope}. Negative reviews per theme over the last 4 complete weeks
          {h && `, ${formatRange(h.period.start, h.period.end)}`}. A review with two themes counts
          once under each.
        </p>
      </div>
      {h && h.untagged_count > 0 && (
        <p className="notice warn" role="status">
          <strong>{h.untagged_count} reviews in this window are not tagged yet.</strong> Counts may
          rise when tagging finishes.
        </p>
      )}
      {state.status === "loading" && (
        <section aria-busy="true">
          <span className="skeleton list-skeleton" aria-hidden="true" />
        </section>
      )}
      {state.status === "error" && (
        <section>
          <div className="error" role="alert">
            <p>
              The heatmap could not load because the server did not answer. Check that OutletOwl is
              running, then reload.
            </p>
          </div>
          <button className="btn btn-secondary" type="button" onClick={reload}>
            Reload heatmap
          </button>
        </section>
      )}
      {empty && (
        <section className="empty">
          <h2>No negative tagged reviews in the last 4 weeks</h2>
          <p>
            Import reviews or wait for tagging to finish; the heatmap fills as reviews are tagged.
          </p>
          {isAdmin && (
            <Link className="btn btn-primary" to="/import">
              Import reviews
            </Link>
          )}
        </section>
      )}
      {h && !empty && (
        <section>
          <TableScroll label="Negative reviews by outlet and theme">
            <table className="data heatmap">
              <caption className="sr-only">
                Negative reviews by outlet and theme, last 4 weeks
              </caption>
              <thead>
                <tr>
                  <th scope="col">Outlet</th>
                  {h.themes.map((t) => (
                    <th key={t.code} scope="col" className="num">
                      {t.label}
                    </th>
                  ))}
                  <th scope="col" className="num">
                    Total
                  </th>
                </tr>
              </thead>
              <tbody>
                {h.rows.map((r) => (
                  <tr key={r.outlet.id}>
                    <th scope="row">{r.outlet.name}</th>
                    {h.themes.map((t) => {
                      const n = r.counts[t.code] ?? 0;
                      return (
                        <td key={t.code} className={`num heat heat-${level(n)}`}>
                          <Link
                            to={reviewsLink({
                              outlet: isAdmin ? r.outlet.id : undefined,
                              theme: t.code,
                              sentiment: "negative",
                              from: h.period.start,
                              to: h.period.end,
                            })}
                            aria-label={`${r.outlet.name}, ${t.label}: ${n} negative reviews`}
                          >
                            {n}
                          </Link>
                        </td>
                      );
                    })}
                    <td className="num">{r.total}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
          <ul className="legend" aria-label="Colour scale">
            {buckets.map((b, i) => (
              <li key={b.label}>
                <span className={`swatch heat-${i}`} aria-hidden="true" />
                {b.label}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
