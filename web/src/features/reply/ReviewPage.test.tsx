import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  adminMe,
  errorReply,
  managerMe,
  stubFetch,
  type Reply as StubReply,
} from "../../test/fetch-stub";
import { review, statusReply } from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";

const draft = (text: string, status = "draft") => ({
  status,
  draft_text: status === "draft" ? text : null,
  prompt_version: 1,
  reply_text: text,
  edited: false,
  replied_by: status === "replied" ? { id: 2, name: "Arjun Mehta" } : null,
  replied_at: status === "replied" ? "2026-10-04T05:50:00Z" : null,
  updated_at: "2026-10-04T05:41:07.512345Z",
});

const detail = (canReply: boolean, reply: unknown = null): StubReply => ({
  status: 200,
  body: {
    ...review(7, "Waited an hour and the dal was cold"),
    reply,
    outlet_managers: [{ id: 2, name: "Arjun Mehta" }],
    can_reply: canReply,
  },
});

const base = { "GET /api/v1/tagging/status": statusReply() };

describe("ReviewPage", () => {
  // AC-US-00-002-1: opening a review with no reply drafts one and shows it.
  it("a manager gets a draft on open, edits it and marks it replied", async () => {
    let sent: Record<string, unknown> = {};
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/7": detail(true),
      "POST /api/v1/reviews/7/draft": {
        status: 200,
        body: draft("Hi Sunita, sorry for the wait."),
      },
      "POST /api/v1/reviews/7/replied": (init) => {
        sent = JSON.parse(String(init.body)) as Record<string, unknown>;
        return { status: 200, body: draft("Hi Sunita, sorry. We are fixing it.", "replied") };
      },
    });
    renderApp("/reviews/7");

    const box = await screen.findByLabelText(/Reply to/);
    expect(box).toHaveValue("Hi Sunita, sorry for the wait.");
    fireEvent.change(box, { target: { value: "Hi Sunita, sorry. We are fixing it." } });
    fireEvent.click(screen.getByRole("button", { name: "Mark as replied" }));

    expect(await screen.findByText("Replied by Arjun Mehta on 4 Oct 2026")).toBeInTheDocument();
    expect(sent).toEqual({
      reply_text: "Hi Sunita, sorry. We are fixing it.",
      based_on_updated_at: "2026-10-04T05:41:07.512345Z",
    });
  });

  // AC-US-00-002-7: drafting unavailable, the manager writes by hand.
  it("budget stop: says drafting is unavailable and lets the manager write and save", async () => {
    let saved: Record<string, unknown> = {};
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/7": detail(true),
      "POST /api/v1/reviews/7/draft": errorReply(
        503,
        "budget_exhausted",
        "Drafting is unavailable. The model budget is used up. Write the reply yourself.",
      ),
      "PUT /api/v1/reviews/7/reply": (init) => {
        saved = JSON.parse(String(init.body)) as Record<string, unknown>;
        return { status: 200, body: { ...draft("Sorry, Sunita."), draft_text: null } };
      },
    });
    renderApp("/reviews/7");

    expect(await screen.findByRole("alert")).toHaveTextContent("model budget is used up");
    fireEvent.change(screen.getByLabelText(/Reply to/), { target: { value: "Sorry, Sunita." } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(saved).toEqual({ reply_text: "Sorry, Sunita.", based_on_updated_at: null });
  });

  it("a conflict says so and offers a reload", async () => {
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/7": detail(true, draft("Hi")),
      "PUT /api/v1/reviews/7/reply": errorReply(
        409,
        "reply_changed",
        "This reply changed since you opened it.",
      ),
    });
    renderApp("/reviews/7");

    fireEvent.click(await screen.findByRole("button", { name: "Save draft" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("changed since you opened it");
    expect(screen.getByRole("button", { name: "Reload the reply" })).toBeInTheDocument();
  });

  it("reloading after a conflict keeps the typed text and saves on the newer reply", async () => {
    let sent: Record<string, unknown> = {};
    let puts = 0;
    let gets = 0;
    const newer = { ...draft("Hi from the other tab"), updated_at: "2026-10-04T06:00:00Z" };
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/7": () => {
        gets += 1;
        return detail(true, gets === 1 ? draft("Hi") : newer);
      },
      "PUT /api/v1/reviews/7/reply": (init) => {
        puts += 1;
        sent = JSON.parse(String(init.body)) as Record<string, unknown>;
        return puts === 1
          ? errorReply(409, "reply_changed", "This reply changed since you opened it.")
          : { status: 200, body: { ...newer, reply_text: "My long edit" } };
      },
    });
    renderApp("/reviews/7");

    const box = await screen.findByLabelText(/Reply to/);
    fireEvent.change(box, { target: { value: "My long edit" } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reload the reply" }));
    expect(await screen.findByText(/Your text is kept/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Reply to/)).toHaveValue("My long edit");

    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(sent).toEqual({
      reply_text: "My long edit",
      based_on_updated_at: "2026-10-04T06:00:00Z",
    });
  });

  it("a hand-written reply whose approval failed is retried on the stored row", async () => {
    const bases: unknown[] = [];
    let marks = 0;
    let puts = 0;
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/7": detail(true),
      "POST /api/v1/reviews/7/draft": errorReply(
        503,
        "model_unavailable",
        "Drafting is unavailable.",
      ),
      "PUT /api/v1/reviews/7/reply": () => {
        puts += 1;
        return { status: 200, body: draft("Sorry.") };
      },
      "POST /api/v1/reviews/7/replied": (init) => {
        marks += 1;
        bases.push((JSON.parse(String(init.body)) as Record<string, unknown>).based_on_updated_at);
        return marks === 1
          ? errorReply(500, "internal", "Something went wrong.")
          : { status: 200, body: draft("Sorry.", "replied") };
      },
    });
    renderApp("/reviews/7");

    const box = await screen.findByLabelText(/Reply to/);
    fireEvent.change(box, { target: { value: "Sorry." } });
    fireEvent.click(screen.getByRole("button", { name: "Mark as replied" }));
    expect(await screen.findByText(/Something went wrong/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mark as replied" }));
    expect(await screen.findByText(/Replied by Arjun Mehta/)).toBeInTheDocument();
    expect(bases).toEqual(["2026-10-04T05:41:07.512345Z", "2026-10-04T05:41:07.512345Z"]);
    expect(puts).toBe(1);
  });

  // AC-US-00-003-3: the admin reads; no draft is requested and no action shows.
  it("the brand admin reads the reply and is told who can reply", async () => {
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/reviews/7": detail(false, draft("Hi Sunita")),
    });
    renderApp("/reviews/7");

    expect(await screen.findByText("Hi Sunita")).toBeInTheDocument();
    expect(screen.getByText(/Only Arjun Mehta, the outlet's manager/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Mark as replied" })).not.toBeInTheDocument();
  });

  it("a review outside the caller's outlet is not found", async () => {
    stubFetch({
      ...base,
      "GET /api/v1/me": { status: 200, body: managerMe },
      "GET /api/v1/reviews/9": errorReply(404, "not_found", "No such review."),
    });
    renderApp("/reviews/9");

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "no such review among the ones you can see",
      ),
    );
  });
});
