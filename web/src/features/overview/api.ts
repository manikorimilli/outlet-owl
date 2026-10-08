import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type Movers = components["schemas"]["Movers"];
export type Trends = components["schemas"]["Trends"];
export type TrendWeek = components["schemas"]["TrendWeek"];

export const getMovers = () => apiFetch<Movers>("/dashboard/movers");
export const getTrends = () => apiFetch<Trends>("/dashboard/trends");

// negativeShare is the week's negative reviews as a share of its tagged
// reviews, the sentiment trend's form (phase 4 LLD); null with none tagged.
export function negativeShare(w: TrendWeek): number | null {
  const tagged = w.positive + w.neutral + w.negative;
  return tagged === 0 ? null : w.negative / tagged;
}
