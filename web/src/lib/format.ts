// Dates arrive as YYYY-MM-DD calendar dates in the brand timezone; they are
// shown as written, never shifted by the browser's zone.
const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

function parts(d: string): [number, number, number] {
  const [y = 0, m = 1, day = 1] = d.split("-").map(Number);
  return [y, m, day];
}

// formatDate shows "3 Oct 2026".
export function formatDate(d: string): string {
  const [y, m, day] = parts(d);
  return `${day} ${months[m - 1]} ${y}`;
}

// formatDay shows "3 Oct", for chart axes where the year is in the title.
export function formatDay(d: string): string {
  const [, m, day] = parts(d);
  return `${day} ${months[m - 1]}`;
}

// formatInstant shows an instant as its date in the brand timezone, so a
// reply approved at 01:30 IST on 5 Oct reads "5 Oct 2026", not UTC's 4 Oct.
export function formatInstant(iso: string, timeZone: string): string {
  // en-CA writes YYYY-MM-DD; formatDate then spells it like every other date.
  const day = new Intl.DateTimeFormat("en-CA", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(iso));
  return formatDate(day);
}

// formatRange shows "28 Sep to 4 Oct 2026", or "21 to 27 Sep" with year
// false.
export function formatRange(start: string, end: string, year = true): string {
  const [ys, ms, ds] = parts(start);
  const [ye, me, de] = parts(end);
  const tail = year ? ` ${ye}` : "";
  if (ms === me && ys === ye) {
    return `${ds} to ${de} ${months[me - 1]}${tail}`;
  }
  return `${ds} ${months[ms - 1]}${ys !== ye ? ` ${ys}` : ""} to ${de} ${months[me - 1]}${tail}`;
}

export const reasonLabels: Record<string, string> = {
  food_safety: "Food safety",
  harassment: "Harassment",
  legal_threat: "Legal threat",
};

// sentimentBadges is the badge colour per sentiment: green, grey, burnt orange.
export const sentimentBadges: Record<string, string> = {
  positive: "ok",
  neutral: "neutral",
  negative: "neg",
};

export const sentimentLabels: Record<string, string> = {
  positive: "Positive",
  neutral: "Neutral",
  negative: "Negative",
};
