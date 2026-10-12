import { describe, expect, it } from "vitest";
import type { Trends, TrendWeek } from "./api";
import { brandWeeks, changeWords, niceMax, percent, weekStats } from "./summary";

const wk = (over: Partial<TrendWeek> = {}): TrendWeek => ({
  week_start: "2026-09-28",
  review_count: 10,
  average_rating: 4,
  positive: 6,
  neutral: 2,
  negative: 2,
  untagged: 0,
  replied: 5,
  ...over,
});

describe("brandWeeks", () => {
  it("sums outlets per week and weights the rating by review count", () => {
    const trends: Trends = {
      weeks: ["2026-09-28"],
      outlets: [
        { outlet: { id: 1, name: "A" }, weeks: [wk({ review_count: 30, average_rating: 4 })] },
        { outlet: { id: 2, name: "B" }, weeks: [wk({ review_count: 10, average_rating: 2 })] },
      ],
    };
    const [w] = brandWeeks(trends);
    expect(w).toMatchObject({ review_count: 40, positive: 12, negative: 4, replied: 10 });
    // (30 * 4 + 10 * 2) / 40 = 3.5, not the plain mean of 3.
    expect(w?.average_rating).toBe(3.5);
  });

  it("leaves the rating empty when no outlet had reviews that week", () => {
    const trends: Trends = {
      weeks: ["2026-09-28"],
      outlets: [
        {
          outlet: { id: 1, name: "A" },
          weeks: [
            wk({ review_count: 0, average_rating: null, positive: 0, neutral: 0, negative: 0 }),
          ],
        },
      ],
    };
    expect(brandWeeks(trends)[0]?.average_rating).toBeNull();
  });
});

describe("weekStats", () => {
  it("compares the latest week with the one before", () => {
    const s = weekStats(
      wk({ review_count: 12, average_rating: 3.4, positive: 4, neutral: 0, negative: 8 }),
      wk({ review_count: 10, average_rating: 4.1, positive: 7, neutral: 0, negative: 3 }),
    );
    expect(s.reviewsChange).toBe(2);
    expect(s.ratingChange).toBe(-0.7);
    // 8 of 12 tagged is 67%, 3 of 10 is 30%: up 37 points.
    expect(s.negativeChange).toBe(37);
  });

  it("has no changes without an earlier week, and no share with nothing tagged", () => {
    const s = weekStats(wk({ positive: 0, neutral: 0, negative: 0, untagged: 10 }), undefined);
    expect(s.reviewsChange).toBeNull();
    expect(s.negativeShare).toBeNull();
  });
});

describe("helpers", () => {
  it("writes changes as words", () => {
    expect(changeWords(3)).toBe("up 3");
    expect(changeWords(-0.2)).toBe("down 0.2");
    expect(changeWords(0)).toBe("no change");
    expect(changeWords(null)).toBeNull();
    expect(percent(0.534)).toBe("53%");
  });

  it("rounds the chart top to a round number", () => {
    expect(niceMax(0)).toBe(5);
    expect(niceMax(13)).toBe(15);
    expect(niceMax(42)).toBe(50);
    expect(niceMax(61)).toBe(80);
  });
});
