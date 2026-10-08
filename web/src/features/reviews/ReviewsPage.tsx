import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";
import { useSession } from "../../app/session-context";
import { asApiError, type ApiError } from "../../lib/api";
import { formatRange } from "../../lib/format";
import { useLoad } from "../../lib/use-load";
import { listOutlets, type OutletSummary } from "../outlets/api";
import {
  apiQuery,
  listReviews,
  listThemes,
  readFilters,
  writeFilters,
  type Filters,
  type ReviewSummary,
  type Theme,
} from "./api";
import { ReviewTable } from "./ReviewTable";

type More = {
  key: string;
  items: ReviewSummary[];
  next: string | null;
  busy: boolean;
  error: ApiError | null;
};

// ReviewsPage is S-04: search and filters kept in the URL, a ruled table, and
// a "Load more" button over the API's cursor.
export function ReviewsPage() {
  const { session } = useSession();
  const me = session.status === "signed-in" ? session.me : null;
  const isAdmin = me?.role === "brand_admin";
  const [params, setParams] = useSearchParams();
  const filters = readFilters(params);
  const key = apiQuery(filters);
  const [page, reload] = useLoad(key, () => listReviews(filters));
  const [themes] = useLoad("themes", listThemes);
  const [outlets] = useLoad(isAdmin ? "outlets" : "none", () =>
    isAdmin ? listOutlets() : Promise.resolve<OutletSummary[]>([]),
  );
  const [more, setMore] = useState<More>({ key, items: [], next: null, busy: false, error: null });
  const [draft, setDraft] = useState({ key, q: filters.q ?? "" });

  // A new filter set starts from its first page and its own search box text.
  const extra = more.key === key ? more : { key, items: [], next: null, busy: false, error: null };
  const q = draft.key === key ? draft.q : (filters.q ?? "");

  function update(next: Filters) {
    setParams(writeFilters(next), { replace: true });
  }

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    update({ ...filters, q: q.trim() || undefined });
  }

  async function loadMore(cursor: string) {
    setMore({ ...extra, busy: true, error: null });
    try {
      const res = await listReviews(filters, cursor);
      setMore({
        key,
        items: [...extra.items, ...res.data],
        next: res.page.next_cursor,
        busy: false,
        error: null,
      });
    } catch (err) {
      setMore({ ...extra, busy: false, error: asApiError(err) });
    }
  }

  const themeList: Theme[] = themes.status === "ready" ? themes.data : [];
  const active = Object.keys(filters).length > 0;
  const title = isAdmin || !me?.outlet ? "Reviews" : `Reviews: ${me.outlet.name}`;

  if (page.status === "error" && page.error.code === "not_found" && me?.outlet) {
    return (
      <div className="content">
        <div className="page-head">
          <h1>{title}</h1>
        </div>
        <section className="empty">
          <p>You can only see reviews for {me.outlet.name}. That link points to another outlet.</p>
          <Link className="btn btn-primary" to="/reviews">
            Show {me.outlet.name} reviews
          </Link>
        </section>
      </div>
    );
  }

  const ready = page.status === "ready" ? page.data : null;
  const rows = ready ? [...ready.data, ...extra.items] : [];
  const next = extra.items.length > 0 ? extra.next : (ready?.page.next_cursor ?? null);
  const empty = ready !== null && ready.total === 0;

  return (
    <div className="content">
      <div className="page-head">
        <h1>
          <span className="hl">{title}</span>
        </h1>
        {ready && (
          <p className="muted">
            <span className="num">{ready.total}</span> {ready.total === 1 ? "review" : "reviews"},
            newest first.
          </p>
        )}
        {(filters.from || filters.to) && (
          <p>
            {filters.from && filters.to
              ? `Showing reviews from ${formatRange(filters.from, filters.to)}.`
              : filters.from
                ? `Showing reviews from ${filters.from} on.`
                : `Showing reviews up to ${filters.to}.`}{" "}
            <button
              className="btn btn-secondary btn-small"
              type="button"
              onClick={() => update({ ...filters, from: undefined, to: undefined })}
            >
              Show all dates
            </button>
          </p>
        )}
      </div>

      {!(empty && !active) && (
        <section aria-label="Search and filters">
          <form className="filters" onSubmit={search} role="search">
            <label className="f-search">
              Search reviews
              <input
                type="search"
                value={q}
                maxLength={200}
                onChange={(e) => setDraft({ key, q: e.target.value })}
                placeholder="Words in the review or the reviewer name"
              />
            </label>
            {isAdmin && (
              <label>
                Outlet
                <select
                  value={filters.outlet ?? ""}
                  onChange={(e) =>
                    update({
                      ...filters,
                      outlet: e.target.value ? Number(e.target.value) : undefined,
                    })
                  }
                >
                  <option value="">All outlets</option>
                  {outlets.status === "ready" &&
                    outlets.data.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.name}
                      </option>
                    ))}
                </select>
              </label>
            )}
            <label>
              Theme
              <select
                value={filters.theme ?? ""}
                onChange={(e) => update({ ...filters, theme: e.target.value || undefined })}
              >
                <option value="">All themes</option>
                {themeList.map((t) => (
                  <option key={t.code} value={t.code}>
                    {t.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Sentiment
              <select
                value={filters.sentiment ?? ""}
                onChange={(e) =>
                  update({
                    ...filters,
                    sentiment: (e.target.value || undefined) as Filters["sentiment"],
                  })
                }
              >
                <option value="">Any</option>
                <option value="positive">Positive</option>
                <option value="neutral">Neutral</option>
                <option value="negative">Negative</option>
              </select>
            </label>
            <label>
              Reply
              <select
                value={filters.reply ?? ""}
                onChange={(e) =>
                  update({ ...filters, reply: (e.target.value || undefined) as Filters["reply"] })
                }
              >
                <option value="">Any</option>
                <option value="none">Not replied</option>
                <option value="draft">Draft ready</option>
                <option value="replied">Replied</option>
              </select>
            </label>
            <label className="check">
              <input
                type="checkbox"
                checked={filters.urgent ?? false}
                onChange={(e) => update({ ...filters, urgent: e.target.checked || undefined })}
              />
              Urgent only
            </label>
            <button className="btn btn-secondary" type="submit">
              Search
            </button>
          </form>
        </section>
      )}

      {page.status === "loading" && (
        <section aria-busy="true">
          <p role="status">Loading reviews</p>
          <span className="skeleton list-skeleton" aria-hidden="true" />
        </section>
      )}
      {page.status === "error" && (
        <section>
          <div className="error" role="alert">
            <p>
              {page.error.status === 422 || page.error.status === 400
                ? `The server refused these filters: ${page.error.message}`
                : "Reviews could not load because the server did not answer. Your filters are kept; check that OutletOwl is running, then try again."}
            </p>
            {page.error.requestId && (
              <p className="request-id">Request id: {page.error.requestId}</p>
            )}
          </div>
          <button className="btn btn-secondary" type="button" onClick={reload}>
            Load reviews again
          </button>{" "}
          {active && (
            <Link className="btn btn-secondary" to="/reviews">
              Clear filters
            </Link>
          )}
        </section>
      )}
      {empty && !active && (
        <section className="empty">
          <h2>No reviews yet</h2>
          <p>Reviews appear here after a CSV import.</p>
          {isAdmin && (
            <Link className="btn btn-primary" to="/import">
              Import reviews
            </Link>
          )}
        </section>
      )}
      {empty && active && (
        <section className="empty">
          <h2>No reviews match</h2>
          <p>
            {filters.q
              ? `Nothing matches "${filters.q}" with these filters.`
              : "Nothing matches these filters."}
          </p>
          <button className="btn btn-secondary" type="button" onClick={() => update({})}>
            Clear filters
          </button>
        </section>
      )}
      {rows.length > 0 && (
        <section>
          <ReviewTable reviews={rows} themes={themeList} showOutlet={isAdmin} />
          {next && (
            <button
              className="btn btn-secondary"
              type="button"
              disabled={extra.busy}
              onClick={() => loadMore(next)}
            >
              {extra.busy ? "Loading" : "Load more"}
            </button>
          )}
          {extra.error && (
            <p className="error" role="alert">
              More reviews could not load. Try again.
            </p>
          )}
        </section>
      )}
    </div>
  );
}
