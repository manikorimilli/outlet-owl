import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, stubFetch, type Reply } from "../../test/fetch-stub";
import { renderApp } from "../../test/render-app";
import type { OutletSummary } from "./api";

function outlet(id: number, name: string, managers: string[] = ["Neha Kulkarni"]): OutletSummary {
  return {
    id,
    name,
    managers: managers.map((m, i) => ({ id: 100 + id * 10 + i, name: m })),
    review_count: 0,
    untagged_count: 0,
    created_at: "2026-10-07T06:00:00Z",
  };
}

const list = (...outlets: OutletSummary[]): Reply => ({ status: 200, body: { data: outlets } });

// replies answers each call with the next reply in turn.
function replies(...answers: (Reply | Promise<Reply>)[]) {
  let call = 0;
  return () => answers[Math.min(call++, answers.length - 1)] as Reply | Promise<Reply>;
}

function later() {
  let resolve: (r: Reply) => void = () => undefined;
  const promise = new Promise<Reply>((r) => (resolve = r));
  return { promise, resolve };
}

function rowNames() {
  const table = screen.getByRole("table", { name: "Outlets" });
  return within(table)
    .getAllByRole("rowheader")
    .map((th) => th.textContent);
}

async function addOutlet(name: string) {
  fireEvent.change(await screen.findByLabelText("Outlet name"), { target: { value: name } });
  fireEvent.click(screen.getByRole("button", { name: "Add outlet" }));
}

describe("OutletsPage", () => {
  it("the add form is usable while the list loads", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": () => later().promise,
    });

    renderApp("/outlets");

    expect(await screen.findByText("Loading outlets")).toBeInTheDocument();
    expect(screen.getByLabelText("Outlet name")).toBeEnabled();
    expect(screen.getByRole("button", { name: "Add outlet" })).toBeEnabled();
  });

  it("an empty list shows the first-run copy", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: adminMe }, "GET /api/v1/outlets": list() });

    renderApp("/outlets");

    expect(await screen.findByRole("heading", { name: "No outlets yet" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "Add each outlet once, using the same name your CSV files use. Then import reviews.",
      ),
    ).toBeInTheDocument();
  });

  it("a failed load shows the reload message and Reload outlets refetches", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": replies(
        errorReply(500, "internal", "Something failed."),
        list(outlet(1, "Koramangala")),
      ),
    });
    renderApp("/outlets");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Outlets could not load because the server did not answer.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Reload outlets" }));

    expect(await screen.findByRole("rowheader", { name: "Koramangala" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("an outlet with no managers shows the no-manager badge", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": list(outlet(1, "Electronic City", []), outlet(2, "Indiranagar")),
    });

    renderApp("/outlets");

    const row = (await screen.findByRole("rowheader", { name: "Electronic City" })).closest("tr");
    expect(row).toHaveTextContent("No manager: add one to the users file");
    expect(screen.getByRole("rowheader", { name: "Indiranagar" }).closest("tr")).toHaveTextContent(
      "Neha Kulkarni",
    );
  });

  it("an added outlet appears in the table in name order", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": replies(
        list(outlet(1, "Indiranagar"), outlet(2, "Koramangala")),
        list(outlet(3, "HSR Layout", []), outlet(1, "Indiranagar"), outlet(2, "Koramangala")),
      ),
      "POST /api/v1/outlets": { status: 201, body: outlet(3, "HSR Layout", []) },
    });
    renderApp("/outlets");
    await screen.findByRole("rowheader", { name: "Koramangala" });

    await addOutlet("HSR Layout");

    const added = await screen.findByRole("rowheader", { name: "HSR Layout" });
    expect(rowNames()).toEqual(["HSR Layout", "Indiranagar", "Koramangala"]);
    expect(added).toHaveFocus();
    expect(screen.getByLabelText("Outlet name")).toHaveValue("");
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("an outlet added while the first load is in flight still appears", async () => {
    const first = later();
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": replies(first.promise, list(outlet(1, "Koramangala"))),
      "POST /api/v1/outlets": { status: 201, body: outlet(1, "Koramangala") },
    });
    renderApp("/outlets");
    await screen.findByText("Loading outlets");

    await addOutlet("Koramangala");
    await screen.findByRole("rowheader", { name: "Koramangala" });
    first.resolve(list());

    await new Promise((r) => setTimeout(r, 0));
    expect(screen.getByRole("rowheader", { name: "Koramangala" })).toBeInTheDocument();
    expect(screen.queryByText("No outlets yet")).not.toBeInTheDocument();
  });

  it("an outlet name is rendered as text", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": list(outlet(1, "<b>Bold</b> & Co")),
    });

    renderApp("/outlets");

    const cell = await screen.findByRole("rowheader", { name: "<b>Bold</b> & Co" });
    expect(cell.querySelector("b")).toBeNull();
  });
});
