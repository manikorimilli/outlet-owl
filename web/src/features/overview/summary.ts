import { negativeShare, type Trends, type TrendWeek } from "./api";

// brandWeeks sums the outlets in scope into one row per week: all outlets for
// the brand admin, the manager's own outlet for a manager (the server scopes
// the trends). The average rating is weighted by each outlet's review count.
export function brandWeeks(trends: Trends): TrendWeek[] {
  return trends.weeks.map((week_start) => {
    const sum: TrendWeek = {
      week_start,
      review_count: 0,
      average_rating: null,
      positive: 0,
      neutral: 0,
      negative: 0,
      untagged: 0,
      replied: 0,
    };
    let ratingTotal = 0;
    let rated = 0;
    for (const { weeks } of trends.outlets) {
      const w = weeks.find((x) => x.week_start === week_start);
      if (!w) continue;
      sum.review_count += w.review_count;
      sum.positive += w.positive;
      sum.neutral += w.neutral;
      sum.negative += w.negative;
      sum.untagged += w.untagged;
      sum.replied += w.replied;
      if (w.average_rating !== null && w.review_count > 0) {
        ratingTotal += w.average_rating * w.review_count;
        rated += w.review_count;
      }
    }
    sum.average_rating = rated === 0 ? null : Math.round((ratingTotal / rated) * 10) / 10;
    return sum;
  });
}

export type WeekStats = {
  reviews: number;
  reviewsChange: number | null;
  rating: number | null;
  ratingChange: number | null;
  negativeShare: number | null;
  // Percentage points, rounded to whole points.
  negativeChange: number | null;
  repliedShare: number | null;
};

// weekStats is the headline numbers for the latest complete week, each with
// its change from the week before when both weeks have a value.
export function weekStats(cur: TrendWeek, prev: TrendWeek | undefined): WeekStats {
  const share = negativeShare(cur);
  const prevShare = prev ? negativeShare(prev) : null;
  return {
    reviews: cur.review_count,
    reviewsChange: prev ? cur.review_count - prev.review_count : null,
    rating: cur.average_rating,
    ratingChange:
      prev?.average_rating != null && cur.average_rating !== null
        ? Math.round((cur.average_rating - prev.average_rating) * 10) / 10
        : null,
    negativeShare: share,
    negativeChange:
      share !== null && prevShare !== null ? Math.round((share - prevShare) * 100) : null,
    repliedShare: cur.review_count === 0 ? null : cur.replied / cur.review_count,
  };
}

// changeWords writes a change as words, never colour alone: "up 3",
// "down 0.2", "no change"; null when there is nothing to compare.
export function changeWords(n: number | null, unit = ""): string | null {
  if (n === null) return null;
  if (n === 0) return "no change";
  return `${n > 0 ? "up" : "down"} ${Math.abs(n)}${unit}`;
}

// percent shows a share as a whole percent, or "none" without one.
export function percent(v: number | null): string {
  return v === null ? "none" : `${Math.round(v * 100)}%`;
}

// niceMax rounds the tallest bar up to a step of 5, 10 or 20 so the grid
// lines land on round numbers.
export function niceMax(n: number): number {
  if (n <= 0) return 5;
  const step = n <= 20 ? 5 : n <= 60 ? 10 : 20;
  return Math.ceil(n / step) * step;
}
