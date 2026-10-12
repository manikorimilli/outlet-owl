import type { ReactNode } from "react";
import { changeWords, percent, type WeekStats } from "./summary";

type Tone = "good" | "bad" | "flat";

// tone says whether a change is good news; up is good for a rating and bad
// for the negative share. The words carry the meaning; colour only repeats it.
function tone(n: number | null, upIsGood: boolean): Tone {
  if (n === null || n === 0) return "flat";
  return n > 0 === upIsGood ? "good" : "bad";
}

function vsLastWeek(change: string | null): string {
  return change === null ? "no earlier week to compare" : `${change} vs last week`;
}

function Tile({
  label,
  value,
  note,
  noteTone = "flat",
}: {
  label: string;
  value: ReactNode;
  note: string;
  noteTone?: Tone;
}) {
  return (
    <div className="stat-tile">
      <dt>{label}</dt>
      <dd className="stat-value num">{value}</dd>
      <dd className={`stat-note ${noteTone}`}>{note}</dd>
    </div>
  );
}

// StatTiles is the headline row of S-02: the latest complete week's reviews,
// rating, negative share and replies, each against the week before.
export function StatTiles({ stats }: { stats: WeekStats }) {
  return (
    <dl className="stat-tiles span-all" aria-label="This week at a glance">
      <Tile
        label="Reviews"
        value={String(stats.reviews)}
        note={vsLastWeek(changeWords(stats.reviewsChange))}
      />
      <Tile
        label="Average rating"
        value={
          stats.rating === null ? (
            "none"
          ) : (
            <>
              {stats.rating.toFixed(1)}
              <span className="stat-unit"> / 5</span>
            </>
          )
        }
        note={vsLastWeek(changeWords(stats.ratingChange))}
        noteTone={tone(stats.ratingChange, true)}
      />
      <Tile
        label="Negative"
        value={percent(stats.negativeShare)}
        note={vsLastWeek(changeWords(stats.negativeChange, " points"))}
        noteTone={tone(stats.negativeChange, false)}
      />
      <Tile label="Replied" value={percent(stats.repliedShare)} note="of this week's reviews" />
    </dl>
  );
}
