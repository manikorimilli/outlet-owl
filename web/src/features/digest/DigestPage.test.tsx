import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, stubFetch } from "../../test/fetch-stub";
import { statusReply } from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";

const sent = {
  id: 7,
  status: "sent",
  week_start: "2026-09-28",
  week_end: "2026-10-04",
  recipient_email: "ritika.rao@example.in",
  untagged_count: 0,
  subject: "Weekly digest, 28 Sep to 4 Oct 2026: Koramangala wait time up 8",
  body: "Biggest mover: Koramangala, wait time.\n<b>not markup</b>",
  sent_at: "2026-10-06T05:12:00Z",
  failure_reason: null,
  created_at: "2026-10-06T05:12:00Z",
};

describe("DigestPage", () => {
  it("generates the digest with a key and shows what was sent, as text", async () => {
    const keys: string[] = [];
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/tagging/status": statusReply(),
      "POST /api/v1/digests": (init) => {
        keys.push(new Headers(init.headers).get("Idempotency-Key") ?? "");
        return { status: 201, body: sent };
      },
    });
    renderApp("/digest");

    fireEvent.click(await screen.findByRole("button", { name: "Generate digest" }));

    expect(await screen.findByRole("heading", { name: sent.subject })).toBeInTheDocument();
    expect(
      screen.getByText(/Sent to ritika.rao@example.in, week of 28 Sep to 4 Oct 2026/),
    ).toBeInTheDocument();
    expect(screen.getByText(/<b>not markup<\/b>/)).toBeInTheDocument();
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/);
  });

  it("MailHog down says what to do", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/tagging/status": statusReply(),
      "POST /api/v1/digests": errorReply(502, "mail_unavailable", "x"),
    });
    renderApp("/digest");

    fireEvent.click(await screen.findByRole("button", { name: "Generate digest" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Start MailHog");
  });
});
