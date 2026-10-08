import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, stubFetch, type Reply } from "../../test/fetch-stub";
import { statusReply, themesReply } from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";
import { expectAccessibleStructure } from "../../test/a11y";

const heatmap = (counts: Record<string, number>, untagged = 0): Reply => ({
  status: 200,
  body: {
    period: { start: "2026-09-07", end: "2026-10-04" },
    themes: (themesReply.body as { data: unknown[] }).data,
    rows: [
      {
        outlet: { id: 1, name: "Koramangala" },
        counts,
        total: Object.values(counts).reduce((a, b) => a + b, 0),
      },
    ],
    untagged_count: untagged,
  },
});

const zero = { food: 0, wait_time: 0, staff: 0, cleanliness: 0, price: 0 };

describe("ThemesPage", () => {
  // AC-US-01-006-1, -3: one column per theme, the count in every cell, a labelled legend.
  it("shows the count in every cell, links each cell, and a legend", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/tagging/status": statusReply(),
      "GET /api/v1/dashboard/heatmap": heatmap({ ...zero, food: 7, wait_time: 19 }),
    });
    renderApp("/themes");

    const table = await screen.findByRole("table", {
      name: /Negative reviews by outlet and theme/,
    });
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((c) => c.textContent),
    ).toEqual(["Outlet", "Food", "Wait time", "Staff", "Cleanliness", "Price", "Total"]);
    const cell = within(table).getByRole("link", {
      name: "Koramangala, Wait time: 19 negative reviews",
    });
    expect(cell).toHaveAttribute(
      "href",
      "/reviews?outlet=1&theme=wait_time&sentiment=negative&from=2026-09-07&to=2026-10-04",
    );
    expect(within(table).getByText("26")).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Colour scale" })).toHaveTextContent("15 or more");
    expect(screen.getByText(/7 Sep to 4 Oct 2026/)).toBeInTheDocument();
    expectAccessibleStructure();
  });

  it("empty and partial", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/tagging/status": statusReply(),
      "GET /api/v1/dashboard/heatmap": heatmap(zero, 12),
    });
    renderApp("/themes");

    expect(
      await screen.findByText(/12 reviews in this window are not tagged yet/),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /No negative tagged reviews/ })).toBeInTheDocument();
  });
});
