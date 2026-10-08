import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, stubFetch } from "../../test/fetch-stub";
import { statusReply } from "../../test/phase4-fixtures";
import { renderApp } from "../../test/render-app";

describe("StatusStrip", () => {
  const cases: [Record<string, unknown>, string][] = [
    [{}, "All reviews tagged"],
    [{ untagged_count: 482 }, "Tagging 482 new reviews"],
    [{ worker: "paused", untagged_count: 7 }, "Tagging paused, 7 waiting"],
    [
      {
        budget: {
          state: "provider_credit_exhausted",
          spent_minor: 1,
          limit_minor: 800,
          currency: "USD",
        },
      },
      "Model budget used up",
    ],
  ];
  for (const [over, text] of cases) {
    it(`shows "${text}"`, async () => {
      stubFetch({
        "GET /api/v1/me": { status: 200, body: adminMe },
        "GET /api/v1/tagging/status": statusReply(over),
        "GET /api/v1/outlets": { status: 200, body: { data: [] } },
      });
      renderApp("/outlets");
      expect(await screen.findByText(text)).toBeInTheDocument();
    });
  }
});
