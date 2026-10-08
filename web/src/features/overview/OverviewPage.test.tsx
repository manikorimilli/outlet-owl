import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, managerMe, stubFetch } from "../../test/fetch-stub";
import {
  moversReply,
  outletsReply,
  page,
  review,
  statusReply,
  trendsReply,
  week,
} from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";
import { apiQuery } from "../reviews/api";

const urgentKey = `GET /api/v1/reviews${apiQuery({ urgent: true, from: week.start, to: week.end })}`;

function routes(over: Record<string, unknown> = {}) {
  return {
    "GET /api/v1/me": { status: 200, body: adminMe },
    "GET /api/v1/dashboard/movers": moversReply(),
    "GET /api/v1/dashboard/trends": trendsReply,
    "GET /api/v1/tagging/status": statusReply(),
    "GET /api/v1/outlets": outletsReply(),
    [urgentKey]: page([review(1488, "खाने में कीड़ा मिला", ["food_safety"])], 3),
    ...over,
  } as Parameters<typeof stubFetch>[0];
}

describe("OverviewPage", () => {
  // AC-US-01-007-3: both weeks' counts and the change are shown.
  it("success: the week line, the top mover sentence and the ranked movers", async () => {
    stubFetch(routes());
    renderApp("/");

    expect(
      await screen.findByText(
        /Latest complete week: 28 Sep to 4 Oct 2026, compared with 21 to 27 Sep/,
      ),
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/Koramangala, wait time: negative reviews up from/),
    ).toBeInTheDocument();
    const table = screen.getByRole("table", { name: /Biggest movers/ });
    const rows = within(table).getAllByRole("row").slice(1);
    expect(rows[0]).toHaveTextContent("KoramangalaWait time311up 8");
    expect(rows[1]).toHaveTextContent("down 2");
  });

  // AC-US-01-008-3: the urgent count is visible and leads to the urgent list in one step.
  it("urgent this week: the count, the reviews as text, and one link to the list", async () => {
    stubFetch(routes());
    renderApp("/");

    expect(await screen.findByRole("heading", { name: "Urgent this week 3" })).toBeInTheDocument();
    expect(screen.getByText("खाने में कीड़ा मिला")).toBeInTheDocument();
    expect(screen.getByText("Urgent: Food safety")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Show all urgent reviews" })).toHaveAttribute(
      "href",
      "/reviews?urgent=true&from=2026-09-28&to=2026-10-04",
    );
  });

  // AC-US-01-005-1, -2: rating and negative share per outlet over 12 weeks.
  it("outlets compared: latest rating, its change and two sparklines", async () => {
    stubFetch(routes());
    renderApp("/");

    const table = await screen.findByRole("table", { name: /Outlets compared/ });
    expect(
      within(table).getByRole("img", { name: "Koramangala rating over 12 weeks, 4.1 to 3.4" }),
    ).toBeInTheDocument();
    expect(
      within(table).getByRole("img", {
        name: "Koramangala negative share over 12 weeks, 30% to 53%",
      }),
    ).toBeInTheDocument();
    expect(within(table).getByText("down 0.7")).toBeInTheDocument();
  });

  it("partial, budget stop and paused notices", async () => {
    stubFetch(
      routes({
        "GET /api/v1/dashboard/movers": moversReply(12),
        "GET /api/v1/tagging/status": statusReply({
          worker: "paused",
          untagged_count: 7,
          budget: {
            state: "budget_exhausted",
            spent_minor: 803,
            limit_minor: 800,
            currency: "USD",
          },
        }),
      }),
    );
    renderApp("/");

    expect(
      await screen.findByText(/12 reviews from these two weeks are not tagged yet/),
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/Model budget used up: USD 8.03 of USD 8.00/),
    ).toBeInTheDocument();
    expect(screen.getByText(/Tagging is switched off/)).toBeInTheDocument();
  });

  it("empty: no reviews yet invites the import", async () => {
    stubFetch(routes({ "GET /api/v1/outlets": outletsReply(0) }));
    renderApp("/");

    expect(await screen.findByRole("heading", { name: "No reviews yet" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Import reviews" })).toHaveAttribute("href", "/import");
  });

  it("error: says what failed and offers a reload", async () => {
    stubFetch(routes({ "GET /api/v1/dashboard/movers": errorReply(500, "internal", "x") }));
    renderApp("/");

    expect(await screen.findByRole("alert")).toHaveTextContent("The overview could not load");
    expect(screen.getByRole("button", { name: "Reload overview" })).toBeInTheDocument();
  });

  // AC-US-00-001-2: a manager's overview names their outlet; the server scopes the data.
  it("manager: titled with their outlet", async () => {
    stubFetch(routes({ "GET /api/v1/me": { status: 200, body: managerMe } }));
    renderApp("/");

    expect(
      await screen.findByRole("heading", { name: "Overview: Indiranagar" }),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("heading", { name: "Your outlet over 12 weeks" }),
    ).toBeInTheDocument();
  });
});
