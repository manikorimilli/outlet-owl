import { useState } from "react";
import { TableScroll } from "../../components/TableScroll";
import { formatDate, formatDay } from "../../lib/format";
import type { TrendWeek } from "./api";

const W = 360;
const H = 240;
const PAD = { top: 24, right: 16, bottom: 28, left: 28 };

// RatingChart is S-02's average rating line over 12 weeks, on its own 1 to 5
// axis (never a second axis on the sentiment chart). Weeks with no reviews
// are gaps. Hover shows the week's rating; a hidden table repeats the data.
export function RatingChart({ weeks }: { weeks: TrendWeek[] }) {
  const [active, setActive] = useState<number | null>(null);
  const plotH = H - PAD.top - PAD.bottom;
  const step = (W - PAD.left - PAD.right) / Math.max(weeks.length - 1, 1);
  const x = (i: number) => PAD.left + step * i;
  const y = (v: number) => PAD.top + plotH - ((v - 1) / 4) * plotH;
  const last = weeks.length - 1;

  // Runs of consecutive rated weeks become separate lines, so a gap stays a gap.
  const runs: string[] = [];
  let run: string[] = [];
  weeks.forEach((w, i) => {
    if (w.average_rating === null) {
      if (run.length) runs.push(run.join(" "));
      run = [];
    } else {
      run.push(`${x(i).toFixed(1)},${y(w.average_rating).toFixed(1)}`);
    }
  });
  if (run.length) runs.push(run.join(" "));
  // The readout shows the hovered week, else the latest one.
  const tip = weeks[active ?? last];

  return (
    <div className="chart">
      {tip && (
        <p className="chart-readout" aria-hidden="true">
          Week of {formatDate(tip.week_start)}:{" "}
          {tip.average_rating === null
            ? "no reviews"
            : `${tip.average_rating.toFixed(1)} from ${tip.review_count} reviews`}
        </p>
      )}
      <TableScroll label="Average rating by week chart">
        <div className="chart-plot narrow" onMouseLeave={() => setActive(null)}>
          <svg viewBox={`0 0 ${W} ${H}`} aria-hidden="true" focusable="false">
            {[1, 2, 3, 4, 5].map((v) => (
              <g key={v}>
                <line className="grid" x1={PAD.left} x2={W - PAD.right} y1={y(v)} y2={y(v)} />
                <text className="axis" x={PAD.left - 8} y={y(v) + 4} textAnchor="end">
                  {v}
                </text>
              </g>
            ))}
            {runs.map((points) => (
              <polyline key={points} className="rating-line" points={points} />
            ))}
            {weeks.map((w, i) => (
              <g key={w.week_start}>
                {w.average_rating !== null && (
                  <circle
                    className={active === i ? "rating-dot on" : "rating-dot"}
                    cx={x(i)}
                    cy={y(w.average_rating)}
                    r="4"
                  />
                )}
                {(i === 0 || i === last || i === Math.floor(last / 2)) && (
                  <text className="axis" x={x(i)} y={H - 8} textAnchor="middle">
                    {formatDay(w.week_start)}
                  </text>
                )}
                <rect
                  className="hit"
                  x={x(i) - step / 2}
                  y={PAD.top}
                  width={step}
                  height={plotH}
                  onMouseEnter={() => setActive(i)}
                />
              </g>
            ))}
          </svg>
        </div>
      </TableScroll>
      <table className="sr-only">
        <caption>Average rating, week by week</caption>
        <thead>
          <tr>
            <th scope="col">Week of</th>
            <th scope="col">Average rating</th>
          </tr>
        </thead>
        <tbody>
          {weeks.map((w) => (
            <tr key={w.week_start}>
              <th scope="row">{formatDate(w.week_start)}</th>
              <td>{w.average_rating === null ? "none" : w.average_rating.toFixed(1)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
