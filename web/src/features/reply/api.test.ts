import { describe, expect, it } from "vitest";
import { stubFetch } from "../../test/fetch-stub";
import { draftReply } from "./api";

describe("draftReply", () => {
  it("shares one request between two asks for the same review", async () => {
    const fetchStub = stubFetch({
      "POST /api/v1/reviews/7/draft": {
        status: 503,
        body: { error: { code: "model_unavailable", message: "x", request_id: "r" } },
      },
    });
    const a = draftReply(7).catch(() => "failed");
    const b = draftReply(7).catch(() => "failed");
    expect(await a).toBe("failed");
    expect(await b).toBe("failed");
    expect(fetchStub).toHaveBeenCalledTimes(1);

    await draftReply(7).catch(() => undefined); // a later ask is a new request
    expect(fetchStub).toHaveBeenCalledTimes(2);
  });
});
