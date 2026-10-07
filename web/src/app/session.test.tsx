import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiFetch } from "../lib/api";
import { adminMe, errorReply, stubFetch, unauthorized } from "../test/fetch-stub";
import { renderApp } from "../test/render-app";

describe("the session", () => {
  it("RequireSession sends a signed-out visitor to /sign-in", async () => {
    stubFetch({ "GET /api/v1/me": unauthorized });

    renderApp("/outlets");

    expect(
      await screen.findByRole("heading", { name: "Sign in to review intelligence" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("/sign-in");
  });

  it("a first visit with no cookie shows no expired notice", async () => {
    stubFetch({ "GET /api/v1/me": unauthorized });

    renderApp("/");

    await screen.findByRole("heading", { name: "Sign in to review intelligence" });
    expect(screen.getByTestId("location").textContent).toBe("/sign-in");
    expect(screen.queryByText(/Your session ended/)).not.toBeInTheDocument();
  });

  it("RequireSession shows the reload error when /me fails with 500", async () => {
    stubFetch({ "GET /api/v1/me": errorReply(500, "internal", "Something failed.") });

    renderApp("/");

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Check that it is running, then reload.");
    expect(alert).toHaveTextContent("Request id: req-test-1");
    expect(screen.getByRole("button", { name: "Reload" })).toBeInTheDocument();
  });

  it("RequireSession shows the reload error when the server does not answer", async () => {
    stubFetch({ "GET /api/v1/me": "network" });

    renderApp("/");

    expect(await screen.findByRole("alert")).toHaveTextContent("OutletOwl did not answer.");
  });

  it("a 401 during use lands on /sign-in?expired=1", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "GET /api/v1/outlets": unauthorized,
    });
    renderApp("/");
    await screen.findByRole("heading", { name: "Overview" });

    await act(() => apiFetch("/outlets").catch(() => undefined));

    expect(
      await screen.findByRole("heading", { name: "Sign in to review intelligence" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/sign-in?expired=1");
    expect(screen.getByRole("status")).toHaveTextContent("Your session ended after 8 hours.");
  });
});
