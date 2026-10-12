import { useEffect, useId, useRef, useState, type CSSProperties } from "react";
import { TableScroll } from "../../components/TableScroll";
import { formatDate, formatDay } from "../../lib/format";
import type { TrendWeek } from "./api";
import { niceMax } from "./summary";

const W = 640;
const H = 240;
const PAD = { top: 24, right: 12, bottom: 28, left: 36 };
const GAP = 2; // surface gap between stacked segments

type Part = "negative" | "neutral" | "positive" | "untagged";
// Bottom to top: negative sits on the baseline so its height compares across weeks.
const parts: { key: Part; label: string }[] = [
  { key: "negative", label: "Negative" },
  { key: "neutral", label: "Neutral" },
  { key: "positive", label: "Positive" },
  { key: "untagged", label: "Not tagged" },
];

// tipPlace puts the tooltip beside the bar, on the side with more room, so it
// never covers the bar or reaches past the chart's edge.
function tipPlace(cx: number): CSSProperties {
  const pct = (cx / W) * 100;
  return pct < 50 ? { left: `calc(${pct}% + 16px)` } : { right: `calc(${100 - pct}% + 16px)` };
}

// SentimentChart is S-02's 12 week stacked bar chart: one bar per week,
// split into negative, neutral and positive reviews, with untagged reviews
// hatched on top. Hover shows the week's counts; a hidden table carries the
// same numbers for screen readers.
export function SentimentChart({ weeks }: { weeks: TrendWeek[] }) {
  const [active, setActive] = useState<number | null>(null);
  const hatch = `hatch-${useId().replace(/[^a-zA-Z0-9]/g, "")}`;
  const anyUntagged = weeks.some((w) => w.untagged > 0);
  const shown = parts.filter((p) => p.key !== "untagged" || anyUntagged);
  const max = niceMax(Math.max(0, ...weeks.map((w) => w.review_count)));
  const plotH = H - PAD.top - PAD.bottom;
  const slot = (W - PAD.left - PAD.right) / Math.max(weeks.length, 1);
  const barW = Math.min(32, slot * 0.6);
  const y = (v: number) => PAD.top + plotH - (v / max) * plotH;
  const last = weeks.length - 1;
  const tip = active === null ? null : weeks[active];
  const plot = useRef<HTMLDivElement>(null);
  // On a narrow screen the chart scrolls; open it at the latest weeks, which
  // are the ones the page is about.
  useEffect(() => {
    const box = plot.current?.parentElement;
    if (box) box.scrollLeft = box.scrollWidth;
  }, []);

  return (
    <div className="chart">
      <ul className="chart-legend" aria-hidden="true">
        {shown.map((p) => (
          <li key={p.key}>
            <span className={`swatch ${p.key}`} />
            {p.label}
          </li>
        ))}
      </ul>
      <TableScroll label="Sentiment by week chart">
        <div className="chart-plot" ref={plot} onMouseLeave={() => setActive(null)}>
          <svg viewBox={`0 0 ${W} ${H}`} aria-hidden="true" focusable="false">
            <defs>
              <pattern
                id={hatch}
                width="6"
                height="6"
                patternUnits="userSpaceOnUse"
                patternTransform="rotate(45)"
              >
                <rect width="6" height="6" className="hatch-bg" />
                <line x1="0" y1="0" x2="0" y2="6" className="hatch-line" />
              </pattern>
            </defs>
            {[0, max / 2, max].map((v) => (
              <g key={v}>
                <line className="grid" x1={PAD.left} x2={W - PAD.right} y1={y(v)} y2={y(v)} />
                <text className="axis" x={PAD.left - 6} y={y(v) + 4} textAnchor="end">
                  {v}
                </text>
              </g>
            ))}
            {weeks.map((w, i) => {
              const x = PAD.left + slot * i + (slot - barW) / 2;
              let base = 0;
              return (
                <g key={w.week_start} className={active === i ? "bar on" : "bar"}>
                  {shown.map((p) => {
                    const v = w[p.key];
                    if (v === 0) return null;
                    const top = y(base + v);
                    const h = y(base) - top - GAP;
                    base += v;
                    return h > 0 ? (
                      <rect
                        key={p.key}
                        className={`seg ${p.key}`}
                        x={x}
                        y={top}
                        width={barW}
                        height={h}
                        rx="2"
                        fill={p.key === "untagged" ? `url(#${hatch})` : undefined}
                      />
                    ) : null;
                  })}
                  {(i === last || i % 2 === last % 2) && (
                    <text className="axis" x={x + barW / 2} y={H - 8} textAnchor="middle">
                      {formatDay(w.week_start)}
                    </text>
                  )}
                  {i === last && (
                    <text
                      className="value"
                      x={x + barW / 2}
                      y={y(w.review_count) - 6}
                      textAnchor="middle"
                    >
                      {w.review_count}
                    </text>
                  )}
                  <rect
                    className="hit"
                    x={PAD.left + slot * i}
                    y={PAD.top}
                    width={slot}
                    height={plotH}
                    onMouseEnter={() => setActive(i)}
                  />
                </g>
              );
            })}
          </svg>
          {tip && active !== null && (
            <div className="chart-tip" style={tipPlace(PAD.left + slot * (active + 0.5))}>
              <strong>Week of {formatDate(tip.week_start)}</strong>
              <span>{tip.review_count} reviews</span>
              {shown.map((p) => (
                <span key={p.key}>
                  <i className={`swatch ${p.key}`} /> {p.label}: {tip[p.key]}
                </span>
              ))}
            </div>
          )}
        </div>
      </TableScroll>
      <table className="sr-only">
        <caption>Reviews by sentiment, week by week</caption>
        <thead>
          <tr>
            <th scope="col">Week of</th>
            {shown.map((p) => (
              <th key={p.key} scope="col">
                {p.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {weeks.map((w) => (
            <tr key={w.week_start}>
              <th scope="row">{formatDate(w.week_start)}</th>
              {shown.map((p) => (
                <td key={p.key}>{w[p.key]}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
