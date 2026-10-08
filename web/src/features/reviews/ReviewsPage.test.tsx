import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, managerMe, stubFetch } from "../../test/fetch-stub";
import { outletsReply, page, review, statusReply, themesReply } from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";
import { apiQuery, readFilters } from "./api";

type Routes = Parameters<typeof stubFetch>[0];

const base = {
  "GET /api/v1/me": { status: 200, body: adminMe },
  "GET /api/v1/tagging/status": statusReply(),
  "GET /api/v1/themes": themesReply,
  "GET /api/v1/outlets": outletsReply(),
};
const list = (q: Parameters<typeof apiQuery>[0]) => `GET /api/v1/reviews${apiQuery(q)}`;

describe("ReviewsPage", () => {
  // AC-US-01-008-4: Devanagari text renders as text, as written.
  it("lists reviews with urgent labels, and review text as text", async () => {
    stubFetch({
      ...base,
      [list({})]: page(
        [
          review(1488, "<b>खाने में कीड़ा मिला</b>", ["food_safety"]),
          { ...review(1490, "ok"), tags: null },
        ],
        2,
      ),
    } as Routes);
    renderApp("/reviews");

    const table = await screen.findByRole("table", { name: "Reviews, newest first" });
    expect(within(table).getByText("<b>खाने में कीड़ा मिला</b>")).toBeInTheDocument();
    expect(within(table).getByText("Urgent: Food safety")).toBeInTheDocument();
    expect(within(table).getAllByText("Not tagged yet")).toHaveLength(2);
  });

  // AC-US-01-008-1, -2: the URL drives the request; a bad value drops only itself.
  it("reads filters from the URL and sends them to the API", async () => {
    stubFetch({
      ...base,
      [list({ q: "wait", theme: "staff", urgent: true })]: page([review(1, "Waited an hour")]),
    } as Routes);
    renderApp("/reviews?q=%20wait%20&theme=staff&urgent=true&sentiment=angry&outlet=x");

    expect(await screen.findByText("Waited an hour")).toBeInTheDocument();
    expect(screen.getByLabelText("Urgent only")).toBeChecked();
    expect(screen.getByLabelText("Search reviews")).toHaveValue("wait");
  });

  it("changing a filter rewrites the URL and reloads", async () => {
    stubFetch({
      ...base,
      [list({})]: page([review(1, "first")]),
      [list({ sentiment: "negative" })]: page([review(2, "only negative")]),
    } as Routes);
    renderApp("/reviews");
    await screen.findByText("first");

    fireEvent.change(screen.getByLabelText("Sentiment"), { target: { value: "negative" } });

    expect(await screen.findByText("only negative")).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/reviews?sentiment=negative");
  });

  it("the search box submits a trimmed search", async () => {
    stubFetch({
      ...base,
      [list({})]: page([review(1, "first")]),
      [list({ q: "dal" })]: page([review(2, "Hair in my dal")]),
    } as Routes);
    renderApp("/reviews");
    await screen.findByText("first");

    fireEvent.change(screen.getByLabelText("Search reviews"), { target: { value: "  dal " } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(await screen.findByText("Hair in my dal")).toBeInTheDocument();
  });

  it("load more appends the next page", async () => {
    stubFetch({
      ...base,
      [list({})]: page([review(2, "newer")], 2, "CUR"),
      "GET /api/v1/reviews?cursor=CUR": page([review(1, "older")], 2),
    } as Routes);
    renderApp("/reviews");

    fireEvent.click(await screen.findByRole("button", { name: "Load more" }));

    expect(await screen.findByText("older")).toBeInTheDocument();
    expect(screen.getByText("newer")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
  });

  it("no results offers to clear the filters; empty invites the import", async () => {
    stubFetch({
      ...base,
      [list({ q: "refund" })]: page([], 0),
      [list({})]: page([], 0),
    } as Routes);
    renderApp("/reviews?q=refund");

    expect(
      await screen.findByText('Nothing matches "refund" with these filters.'),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByRole("heading", { name: "No reviews yet" })).toBeInTheDocument();
  });

  it("a manager has no outlet filter, and another outlet's link says whose reviews they see", async () => {
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      [list({ outlet: 9 })]: errorReply(404, "not_found", "No such outlet."),
    } as Routes);
    renderApp("/reviews?outlet=9");

    expect(await screen.findByText(/You can only see reviews for Indiranagar/)).toBeInTheDocument();
    expect(screen.queryByLabelText("Outlet")).not.toBeInTheDocument();
  });

  it("the urgent list opened from the overview names its week", async () => {
    stubFetch({
      ...base,
      [list({ urgent: true, from: "2026-09-28", to: "2026-10-04" })]: page([
        review(1, "x", ["legal_threat"]),
      ]),
    } as Routes);
    renderApp("/reviews?urgent=true&from=2026-09-28&to=2026-10-04");

    expect(
      await screen.findByText(/Showing urgent reviews from the week of 28 Sep to 4 Oct 2026/),
    ).toBeInTheDocument();
  });

  it("readFilters trims the search and drops bad values one by one", () => {
    expect(
      readFilters(new URLSearchParams("q=%20%20&outlet=-1&from=2026-13&to=2026-10-04&urgent=yes")),
    ).toEqual({ to: "2026-10-04" });
  });
});
