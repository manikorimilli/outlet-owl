import { afterEach, describe, expect, it, vi } from "vitest";
import { errorReply, stubFetch, unauthorized } from "../test/fetch-stub";
import { ApiError, apiFetch, setUnauthorizedHandler } from "./api";

afterEach(() => setUnauthorizedHandler(undefined));

async function failure(call: Promise<unknown>): Promise<ApiError> {
  const err = await call.then(
    () => undefined,
    (e: unknown) => e,
  );
  expect(err).toBeInstanceOf(ApiError);
  return err as ApiError;
}

describe("apiFetch", () => {
  it("parses the error envelope into ApiError", async () => {
    stubFetch({
      "POST /api/v1/outlets": errorReply(422, "validation_failed", "Check the name.", [
        { field: "name", reason: "too_long" },
      ]),
    });

    const err = await failure(apiFetch("/outlets", { method: "POST", body: { name: "x" } }));

    expect(err.status).toBe(422);
    expect(err.code).toBe("validation_failed");
    expect(err.message).toBe("Check the name.");
    expect(err.details).toEqual([{ field: "name", reason: "too_long" }]);
    expect(err.requestId).toBe("req-test-1");
  });

  it("reports a failed fetch as code network", async () => {
    stubFetch({ "GET /api/v1/me": "network" });

    const err = await failure(apiFetch("/me"));

    expect(err.code).toBe("network");
    expect(err.status).toBe(0);
  });

  it("calls the unauthorized handler on 401 unauthorized", async () => {
    stubFetch({ "GET /api/v1/outlets": unauthorized });
    const handler = vi.fn();
    setUnauthorizedHandler(handler);

    await failure(apiFetch("/outlets"));

    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not call the unauthorized handler on 401 invalid_credentials", async () => {
    stubFetch({
      "POST /api/v1/auth/login": errorReply(401, "invalid_credentials", "No match."),
    });
    const handler = vi.fn();
    setUnauthorizedHandler(handler);

    const err = await failure(apiFetch("/auth/login", { method: "POST", body: {} }));

    expect(err.code).toBe("invalid_credentials");
    expect(handler).not.toHaveBeenCalled();
  });

  it("reads a body that is not the envelope as internal, keeping the request id", async () => {
    stubFetch({
      "GET /api/v1/outlets": {
        status: 502,
        body: "Bad Gateway",
        headers: { "X-Request-Id": "r-9" },
      },
    });

    const err = await failure(apiFetch("/outlets"));

    expect(err.code).toBe("internal");
    expect(err.status).toBe(502);
    expect(err.requestId).toBe("r-9");
  });

  it("sends JSON with the cookie and returns undefined for 204", async () => {
    const fetchStub = stubFetch({ "POST /api/v1/auth/logout": { status: 204 } });

    await expect(apiFetch("/auth/logout", { method: "POST" })).resolves.toBeUndefined();
    await apiFetch("/auth/logout", { method: "POST", body: { a: 1 } }).catch(() => undefined);

    const init = fetchStub.mock.calls[1]?.[1];
    expect(init?.credentials).toBe("same-origin");
    expect(init?.headers).toMatchObject({ "Content-Type": "application/json" });
    expect(init?.body).toBe('{"a":1}');
  });
});
