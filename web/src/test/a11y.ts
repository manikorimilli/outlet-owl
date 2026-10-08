import { expect } from "vitest";

// expectAccessibleStructure checks the rules a keyboard or screen-reader
// user needs on a rendered screen (the accessibility gate): every form
// control has a label, no positive tabindex, every scrolling table box is
// focusable and named, icon-only buttons have a name. It fails when it found
// nothing to check, so a screen that renders blank cannot pass.
export function expectAccessibleStructure(root: HTMLElement = document.body): void {
  const controls = [...root.querySelectorAll<HTMLElement>("input, select, textarea")].filter(
    (c) => !(c instanceof HTMLInputElement && c.type === "hidden"),
  );
  const buttons = [...root.querySelectorAll<HTMLElement>("button, a[href]")];
  const scrollers = [...root.querySelectorAll<HTMLElement>(".table-scroll")];
  expect(controls.length + buttons.length + scrollers.length, "nothing to check").toBeGreaterThan(
    0,
  );

  for (const c of controls) {
    const labelled =
      ((c as HTMLInputElement).labels?.length ?? 0) > 0 ||
      !!c.getAttribute("aria-label") ||
      !!c.getAttribute("aria-labelledby");
    expect(labelled, `unlabelled ${c.outerHTML.slice(0, 80)}`).toBe(true);
  }
  for (const b of buttons) {
    const name = (b.textContent ?? "").trim() || b.getAttribute("aria-label") || "";
    expect(name, `unnamed ${b.outerHTML.slice(0, 80)}`).not.toBe("");
  }
  for (const el of root.querySelectorAll<HTMLElement>("[tabindex]")) {
    expect(Number(el.getAttribute("tabindex")), "positive tabindex").toBeLessThanOrEqual(0);
  }
  for (const s of scrollers) {
    expect(s.getAttribute("tabindex"), "scroll box not focusable").toBe("0");
    expect(s.getAttribute("aria-label") ?? "", "scroll box has no name").not.toBe("");
  }
}
