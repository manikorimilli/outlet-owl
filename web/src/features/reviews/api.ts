import { apiFetch } from "../../lib/api";
import type { components, operations } from "../../lib/api-types";

export type ReviewSummary = components["schemas"]["ReviewSummary"];
export type Theme = components["schemas"]["Theme"];
type ReviewPage = operations["listReviews"]["responses"][200]["content"]["application/json"];
type ThemeList = operations["listThemes"]["responses"][200]["content"]["application/json"];

// Filters are what S-04 keeps in the URL. Names are short in the URL and
// mapped to the API's filter[...] parameters here, the one place.
export type Filters = {
  q?: string;
  outlet?: number;
  theme?: string;
  sentiment?: "positive" | "neutral" | "negative";
  urgent?: boolean;
  from?: string;
  to?: string;
};

export function apiQuery(f: Filters, cursor?: string): string {
  const p = new URLSearchParams();
  if (f.q) p.set("q", f.q);
  if (f.outlet !== undefined) p.set("filter[outlet_id]", String(f.outlet));
  if (f.theme) p.set("filter[theme]", f.theme);
  if (f.sentiment) p.set("filter[sentiment]", f.sentiment);
  if (f.urgent) p.set("filter[is_urgent]", "true");
  if (f.from) p.set("filter[review_date][gte]", f.from);
  if (f.to) p.set("filter[review_date][lte]", f.to);
  if (cursor) p.set("cursor", cursor);
  const s = p.toString();
  return s ? `?${s}` : "";
}

export function listReviews(f: Filters, cursor?: string): Promise<ReviewPage> {
  return apiFetch<ReviewPage>(`/reviews${apiQuery(f, cursor)}`);
}

export async function listThemes(): Promise<Theme[]> {
  return (await apiFetch<ThemeList>("/themes")).data;
}

const dateRe = /^\d{4}-\d{2}-\d{2}$/;
const sentiments = ["positive", "neutral", "negative"] as const;

// readFilters parses each URL parameter on its own: a bad value drops only
// itself, never the whole page (a pasted or old link still works).
export function readFilters(p: URLSearchParams): Filters {
  const f: Filters = {};
  const q = p.get("q")?.trim();
  if (q) f.q = q.slice(0, 200);
  const outlet = Number(p.get("outlet"));
  if (Number.isInteger(outlet) && outlet > 0) f.outlet = outlet;
  const theme = p.get("theme");
  if (theme && /^[a-z][a-z0-9_]*$/.test(theme)) f.theme = theme;
  const s = p.get("sentiment");
  const sentiment = sentiments.find((x) => x === s);
  if (sentiment) f.sentiment = sentiment;
  if (p.get("urgent") === "true") f.urgent = true;
  const from = p.get("from");
  if (from && dateRe.test(from)) f.from = from;
  const to = p.get("to");
  if (to && dateRe.test(to)) f.to = to;
  return f;
}

export function writeFilters(f: Filters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.q) p.set("q", f.q);
  if (f.outlet !== undefined) p.set("outlet", String(f.outlet));
  if (f.theme) p.set("theme", f.theme);
  if (f.sentiment) p.set("sentiment", f.sentiment);
  if (f.urgent) p.set("urgent", "true");
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  return p;
}

// reviewsLink is the URL other screens use to open a filtered list.
export function reviewsLink(f: Filters): string {
  const s = writeFilters(f).toString();
  return s ? `/reviews?${s}` : "/reviews";
}
