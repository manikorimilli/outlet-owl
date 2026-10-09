import type { ReactNode } from "react";

// TableScroll is the sideways-scrolling box around a wide table. It takes
// keyboard focus and is a named region, so a keyboard user can scroll a
// table that holds no links (WCAG 2.1.1); label names the table.
export function TableScroll({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="table-scroll" role="region" aria-label={label} tabIndex={0}>
      {children}
    </div>
  );
}
