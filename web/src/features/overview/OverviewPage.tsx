import { Link } from "react-router";
import { useSession } from "../../app/session-context";
import { formatDate, formatRange, reasonLabels } from "../../lib/format";
import { useLoad } from "../../lib/use-load";
import { listOutlets } from "../outlets/api";
import { listReviews, reviewsLink } from "../reviews/api";
import { dollars, getTaggingStatus } from "../status/api";
import { getMovers, getTrends } from "./api";
import { OutletComparison } from "./OutletComparison";

// OverviewPage is S-02: the week line, the top mover in one sentence, the
// ranked movers, the week's urgent reviews and the outlet comparison.
export function OverviewPage() {
  const { session } = useSession();
  const me = session.status === "signed-in" ? session.me : null;
  const isAdmin = me?.role === "brand_admin";
  const [movers, reloadMovers] = useLoad("movers", getMovers);
  const [trends, reloadTrends] = useLoad("trends", getTrends);
  const [status] = useLoad("status", getTaggingStatus);
  const [outlets] = useLoad("outlets", listOutlets);
  const week = movers.status === "ready" ? movers.data.week : null;
  const urgentFilter = week ? { urgent: true, from: week.start, to: week.end } : null;
  const [urgent] = useLoad(urgentFilter ? `urgent-${week?.start}` : "urgent-wait", () =>
    urgentFilter ? listReviews(urgentFilter) : new Promise<never>(() => undefined),
  );

  const title = isAdmin || !me?.outlet ? "Overview" : `Overview: ${me.outlet.name}`;
  const head = (
    <div className="span-all page-head">
      <h1>
        <span className="hl">{title}</span>
      </h1>
      {movers.status === "ready" && (
        <p className="muted">
          Latest complete week: {formatRange(movers.data.week.start, movers.data.week.end)},
          compared with{" "}
          {formatRange(movers.data.previous_week.start, movers.data.previous_week.end, false)}.{" "}
          <span className="num">{movers.data.review_count}</span> reviews this week.
        </p>
      )}
    </div>
  );

  if (movers.status === "error" || trends.status === "error") {
    return (
      <div className="content">
        {head}
        <section>
          <div className="error" role="alert">
            <p>
              The overview could not load because the server did not answer. Check that OutletOwl is
              running, then reload.
            </p>
          </div>
          <button
            className="btn btn-secondary"
            type="button"
            onClick={() => {
              reloadMovers();
              reloadTrends();
            }}
          >
            Reload overview
          </button>
        </section>
      </div>
    );
  }

  const totalReviews =
    outlets.status === "ready" ? outlets.data.reduce((n, o) => n + o.review_count, 0) : null;
  if (totalReviews === 0) {
    return (
      <div className="content">
        {head}
        <section className="empty">
          <h2>No reviews yet</h2>
          <p>
            Import a CSV of reviews for your outlets. Movers, urgent reviews and trends appear here
            once the first complete week is in.
          </p>
          {isAdmin && (
            <Link className="btn btn-primary" to="/import">
              Import reviews
            </Link>
          )}
        </section>
      </div>
    );
  }

  const m = movers.status === "ready" ? movers.data : null;
  const top = m?.movers[0];
  const s = status.status === "ready" ? status.data : null;

  return (
    <div className="content cols-2">
      {head}
      {s && s.budget.state !== "ok" && (
        <p className="notice warn span-all" role="status">
          <strong>
            Model budget used up: USD {dollars(s.budget.spent_minor)} of USD{" "}
            {dollars(s.budget.limit_minor)}.
          </strong>{" "}
          New reviews are not tagged and reply drafts are unavailable. Movers, search and the
          reports still work.{" "}
          {s.untagged_count > 0 && `${s.untagged_count} new reviews are waiting.`}
        </p>
      )}
      {s && s.worker === "paused" && (
        <p className="notice warn span-all" role="status">
          <strong>Tagging is switched off in this installation&apos;s settings.</strong>{" "}
          {s.untagged_count} new reviews are waiting. Ask whoever runs OutletOwl to turn tagging
          back on.
        </p>
      )}
      {m && m.untagged_count > 0 && (
        <p className="notice warn span-all" role="status">
          <strong>{m.untagged_count} reviews from these two weeks are not tagged yet.</strong>{" "}
          Movers and urgent reviews may be incomplete until tagging finishes; this page updates on
          reload.
        </p>
      )}

      <section aria-labelledby="movers-title">
        <h2 id="movers-title">Biggest movers</h2>
        {movers.status === "loading" && (
          <span className="skeleton list-skeleton" aria-hidden="true" />
        )}
        {m && !top && <p className="muted">No outlet and theme changed between these two weeks.</p>}
        {m && top && (
          <>
            <p className="lead">
              {top.outlet.name}, {top.theme.label.toLowerCase()}: negative reviews{" "}
              {top.change > 0 ? "up" : "down"} from{" "}
              <span className="num">{top.previous_count}</span> to{" "}
              <span className="num">{top.current_count}</span>.
            </p>
            <div className="table-scroll">
              <table className="data">
                <caption className="sr-only">
                  Biggest movers, negative reviews by outlet and theme
                </caption>
                <thead>
                  <tr>
                    <th scope="col">Outlet</th>
                    <th scope="col">Theme</th>
                    <th scope="col" className="num">
                      Last week
                    </th>
                    <th scope="col" className="num">
                      This week
                    </th>
                    <th scope="col" className="num">
                      Change
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {m.movers.slice(0, 10).map((x) => (
                    <tr key={`${x.outlet.id}-${x.theme.code}`}>
                      <th scope="row">
                        <Link
                          to={reviewsLink({
                            outlet: isAdmin ? x.outlet.id : undefined,
                            theme: x.theme.code,
                            sentiment: "negative",
                            from: m.previous_week.start,
                            to: m.week.end,
                          })}
                        >
                          {x.outlet.name}
                        </Link>
                      </th>
                      <td>{x.theme.label}</td>
                      <td className="num">{x.previous_count}</td>
                      <td className="num">{x.current_count}</td>
                      <td className="num">
                        {x.change > 0 ? "up" : "down"} {Math.abs(x.change)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="small muted">
              Ranked by the change in negative reviews per outlet and theme. Open a row to read
              those reviews.
            </p>
          </>
        )}
      </section>

      <aside aria-labelledby="urgent-title">
        <h2 id="urgent-title">
          Urgent this week{" "}
          {urgent.status === "ready" && <span className="num">{urgent.data.total}</span>}
        </h2>
        {urgent.status === "ready" && urgent.data.total === 0 && (
          <p className="muted">No urgent reviews this week.</p>
        )}
        {urgent.status === "ready" && (
          <ul className="urgent-list">
            {urgent.data.data.slice(0, 5).map((r) => (
              <li key={r.id}>
                {r.tags?.urgent_reasons.map((reason) => (
                  <span key={reason} className="badge danger">
                    Urgent: {reasonLabels[reason] ?? reason}
                  </span>
                ))}
                <p className="small muted">
                  {r.outlet.name}, {formatDate(r.review_date)}
                </p>
                <p className="review-text clamp">{r.review_text}</p>
              </li>
            ))}
          </ul>
        )}
        {urgentFilter && urgent.status === "ready" && urgent.data.total > 0 && (
          <Link to={reviewsLink(urgentFilter)}>Show all urgent reviews</Link>
        )}
      </aside>

      <section className="span-all" aria-labelledby="compare-title">
        <h2 id="compare-title">{isAdmin ? "Outlets compared" : "Your outlet over 12 weeks"}</h2>
        {trends.status === "loading" && (
          <span className="skeleton list-skeleton" aria-hidden="true" />
        )}
        {trends.status === "ready" && <OutletComparison trends={trends.data} />}
      </section>
    </div>
  );
}
