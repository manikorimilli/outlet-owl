// Sparkline draws one series as a line, no chart library (phase 4 LLD). Null
// points are gaps. label names the first and last value for screen readers,
// as the S-02 mockup does; the line itself is decoration.
export function Sparkline({
  values,
  label,
  max,
}: {
  values: (number | null)[];
  label: string;
  max: number;
}) {
  const w = 96;
  const h = 24;
  const step = values.length > 1 ? w / (values.length - 1) : 0;
  const points = values
    .map((v, i) =>
      v === null ? null : `${(i * step).toFixed(1)},${(h - (v / max) * h).toFixed(1)}`,
    )
    .filter((p): p is string => p !== null)
    .join(" ");
  return (
    <svg
      className="spark"
      viewBox={`0 0 ${w} ${h}`}
      width={w}
      height={h}
      role="img"
      aria-label={label}
    >
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth="1.75" />
    </svg>
  );
}
