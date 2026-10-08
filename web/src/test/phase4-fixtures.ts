// Fixtures for the phase 4 screens: the latest complete week is 28 Sep to 4 Oct 2026.
import type { Reply } from "./fetch-stub";

export const week = { start: "2026-09-28", end: "2026-10-04" };
export const previousWeek = { start: "2026-09-21", end: "2026-09-27" };

export const outletsReply = (reviewCount = 30): Reply => ({
  status: 200,
  body: {
    data: [
      {
        id: 1,
        name: "Koramangala",
        managers: [],
        review_count: reviewCount,
        untagged_count: 0,
        created_at: "2026-09-01T00:00:00Z",
      },
    ],
  },
});

export const statusReply = (over: Record<string, unknown> = {}): Reply => ({
  status: 200,
  body: {
    untagged_count: 0,
    worker: "idle",
    budget: { state: "ok", spent_minor: 57, limit_minor: 800, currency: "USD" },
    ...over,
  },
});

export const moversReply = (untagged = 0): Reply => ({
  status: 200,
  body: {
    week,
    previous_week: previousWeek,
    review_count: 66,
    untagged_count: untagged,
    movers: [
      {
        outlet: { id: 1, name: "Koramangala" },
        theme: { code: "wait_time", label: "Wait time" },
        previous_count: 3,
        current_count: 11,
        change: 8,
      },
      {
        outlet: { id: 2, name: "Indiranagar" },
        theme: { code: "staff", label: "Staff" },
        previous_count: 4,
        current_count: 2,
        change: -2,
      },
    ],
  },
});

const weeks = Array.from({ length: 12 }, (_, i) =>
  new Date(Date.UTC(2026, 6, 13 + 7 * i)).toISOString().slice(0, 10),
);

export const trendsReply: Reply = {
  status: 200,
  body: {
    weeks,
    outlets: [
      {
        outlet: { id: 1, name: "Koramangala" },
        weeks: weeks.map((w, i) => ({
          week_start: w,
          review_count: 10,
          average_rating: i === 11 ? 3.4 : 4.1,
          positive: 5,
          neutral: 2,
          negative: i === 11 ? 8 : 3,
          untagged: 0,
          replied: 0,
        })),
      },
    ],
  },
};

export const review = (id: number, text: string, urgent: string[] = []) => ({
  id,
  outlet: { id: 1, name: "Koramangala" },
  source: "Google",
  review_date: "2026-10-02",
  rating: 1,
  review_text: text,
  reviewer_name: "Sunita Verma",
  tags: {
    themes: ["food"],
    sentiment: "negative",
    is_urgent: urgent.length > 0,
    urgent_reasons: urgent,
    prompt_version: 1,
  },
  reply_status: "none",
});

export const page = (data: unknown[], total = data.length, next: string | null = null): Reply => ({
  status: 200,
  body: { data, page: { next_cursor: next, has_more: next !== null }, total },
});

export const themesReply: Reply = {
  status: 200,
  body: {
    data: [
      { code: "food", label: "Food" },
      { code: "wait_time", label: "Wait time" },
      { code: "staff", label: "Staff" },
      { code: "cleanliness", label: "Cleanliness" },
      { code: "price", label: "Price" },
    ],
  },
};
