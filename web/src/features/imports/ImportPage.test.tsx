import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, stubFetch, type Reply } from "../../test/fetch-stub";
import { renderApp } from "../../test/render-app";
import type { ImportResult } from "./api";

const someOutlets: Reply = {
  status: 200,
  body: {
    data: [
      {
        id: 1,
        name: "Indiranagar",
        managers: [],
        review_count: 0,
        untagged_count: 0,
        created_at: "2026-10-07T06:00:00Z",
      },
    ],
  },
};

function result(over: Partial<ImportResult> = {}): ImportResult {
  return {
    id: 12,
    file_name: "reviews-sept.csv",
    imported_count: 479,
    duplicate_count: 14,
    rejected_count: 0,
    rejections: [],
    created_at: "2026-10-06T05:10:00Z",
    ...over,
  };
}

const csv = () => new File(["outlet,source\n"], "reviews-sept.csv", { type: "text/csv" });

async function chooseAndImport(file = csv()) {
  fireEvent.change(await screen.findByLabelText("CSV file"), { target: { files: [file] } });
  fireEvent.click(screen.getByRole("button", { name: "Import reviews" }));
}

describe("ImportPage", () => {
  it("ready: the file field and the note on weeks", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
    });

    renderApp("/import");

    expect(await screen.findByRole("heading", { name: "Import reviews" })).toBeInTheDocument();
    expect(await screen.findByText(/weeks in the Asia\/Kolkata time zone/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Import reviews" })).toBeEnabled();
  });

  it("no outlets: sends the admin to add outlets first", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": { status: 200, body: { data: [] } },
    });

    renderApp("/import");

    expect(await screen.findByRole("heading", { name: "Add an outlet first" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add outlets" })).toHaveAttribute("href", "/outlets");
    expect(screen.getByRole("button", { name: "Import reviews" })).toBeDisabled();
  });

  it("success: three counts, tagging started, one key per chosen file", async () => {
    const keys: string[] = [];
    let sentFile: unknown;
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": (init) => {
        keys.push(new Headers(init.headers).get("Idempotency-Key") ?? "");
        sentFile = (init.body as FormData).get("file");
        return { status: 201, body: result() };
      },
    });

    renderApp("/import");
    await chooseAndImport();

    const card = await screen.findByRole("region", { name: "reviews-sept.csv imported" });
    expect(within(card).getByText("479")).toBeInTheDocument();
    expect(within(card).getByText("14")).toBeInTheDocument();
    expect(within(card).getByText(/Tagging 479 new reviews/)).toBeInTheDocument();
    expect(keys).toHaveLength(1);
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/);
    expect((sentFile as File).name).toBe("reviews-sept.csv");
  });

  it("a retry after a lost answer reuses the key; a new file gets a new one", async () => {
    const keys: string[] = [];
    let call = 0;
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": (init) => {
        keys.push(new Headers(init.headers).get("Idempotency-Key") ?? "");
        call += 1;
        return call === 1 ? errorReply(500, "internal", "x") : { status: 201, body: result() };
      },
    });

    renderApp("/import");
    await chooseAndImport();
    expect(await screen.findByRole("alert")).toHaveTextContent("something failed on the server");
    fireEvent.click(screen.getByRole("button", { name: "Import reviews" }));
    await screen.findByRole("region", { name: "reviews-sept.csv imported" });

    expect(keys).toHaveLength(2);
    expect(keys[1]).toBe(keys[0]);

    await chooseAndImport(new File(["x"], "october.csv", { type: "text/csv" }));
    await waitFor(() => expect(keys).toHaveLength(3));
    expect(keys[2]).not.toBe(keys[0]);
  });

  it("a refusal clears the file, so a corrected file is read under a new key", async () => {
    const keys: string[] = [];
    let call = 0;
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": (init) => {
        keys.push(new Headers(init.headers).get("Idempotency-Key") ?? "");
        call += 1;
        return call === 1
          ? errorReply(400, "csv_invalid", "This file has no rating column.")
          : { status: 201, body: result() };
      },
    });

    renderApp("/import");
    await chooseAndImport();
    expect(await screen.findByRole("alert")).toHaveTextContent("no rating column");
    fireEvent.click(screen.getByRole("button", { name: "Import reviews" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Choose a CSV file");
    expect(keys).toHaveLength(1);

    await chooseAndImport();
    await screen.findByRole("region", { name: "reviews-sept.csv imported" });
    expect(keys[1]).not.toBe(keys[0]);
  });

  it("the status region announces the result", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": { status: 201, body: result() },
    });

    renderApp("/import");
    await screen.findByRole("heading", { name: "Import reviews" });
    expect(screen.getByRole("status")).toBeEmptyDOMElement();
    await chooseAndImport();

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent(
        "reviews-sept.csv imported: 479 imported, 14 already imported and skipped, 0 rejected.",
      ),
    );
  });

  it("caps the rejected rows at 200 and says how many there are", async () => {
    const rejections = Array.from({ length: 250 }, (_, i) => ({
      row_number: i + 2,
      reason: "Date is empty.",
    }));
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": {
        status: 201,
        body: result({ imported_count: 0, rejected_count: 250, rejections }),
      },
    });

    renderApp("/import");
    await chooseAndImport();

    const table = await screen.findByRole("table", { name: "Rejected rows" });
    expect(within(table).getAllByRole("row")).toHaveLength(201);
    expect(screen.getByText("Showing the first 200 of 250 rejected rows.")).toBeInTheDocument();
  });

  it("importing: the button is disabled and the progress line shows", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": () => new Promise<Reply>(() => undefined),
    });

    renderApp("/import");
    await chooseAndImport();

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("Reading reviews-sept.csv"),
    );
    expect(screen.getByRole("button", { name: "Importing" })).toBeDisabled();
  });

  it("partial: each rejected row with its number and reason, as text", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": {
        status: 201,
        body: result({
          imported_count: 0,
          rejected_count: 2,
          rejections: [
            { row_number: 37, reason: 'Unknown outlet "<b>Koramangla</b>".' },
            { row_number: 240, reason: "Date is empty." },
          ],
        }),
      },
    });

    renderApp("/import");
    await chooseAndImport();

    const table = await screen.findByRole("table", { name: "Rejected rows" });
    const rows = within(table).getAllByRole("row").slice(1);
    expect(rows.map((r) => r.textContent)).toEqual([
      '37Unknown outlet "<b>Koramangla</b>".',
      "240Date is empty.",
    ]);
    expect(screen.queryByText(/Tagging 0 new/)).not.toBeInTheDocument();
  });

  it("a file the server cannot read shows its sentence naming the column", async () => {
    const msg =
      "This file has no rating column, so nothing was imported. The first row must name: outlet, source, date, rating, text, reviewer_name.";
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": errorReply(400, "csv_invalid", msg, [
        { field: "rating", reason: "missing" },
      ]),
    });

    renderApp("/import");
    await chooseAndImport();

    expect(await screen.findByRole("alert")).toHaveTextContent("no rating column");
    expect(screen.getByLabelText("CSV file")).toHaveAttribute("aria-invalid", "true");
  });

  it("a file over 5 MB says so", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": someOutlets,
      "POST /api/v1/imports": errorReply(
        413,
        "file_too_large",
        "The CSV file is over 5 MB. Split it into smaller files and import each.",
      ),
    });

    renderApp("/import");
    await chooseAndImport();

    expect(await screen.findByRole("alert")).toHaveTextContent("over 5 MB");
  });
});
