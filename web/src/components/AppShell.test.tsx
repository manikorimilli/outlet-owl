import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, managerMe, stubFetch } from "../test/fetch-stub";
import { renderApp } from "../test/render-app";

function navLinks() {
  const nav = screen.getByRole("navigation", { name: "Sections" });
  return within(nav)
    .getAllByRole("link")
    .map((a) => a.textContent);
}

describe("AppShell", () => {
  it("the brand admin sees Overview, Outlets and Import in the nav", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: adminMe } });

    renderApp("/outlets");

    await screen.findByRole("navigation", { name: "Sections" });
    expect(navLinks()).toEqual(["Overview", "Outlets", "Import"]);
    expect(screen.getByRole("link", { name: "Outlets" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Overview" })).not.toHaveAttribute("aria-current");
  });

  it("an outlet manager sees Overview only", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: managerMe } });

    renderApp("/");

    await screen.findByRole("navigation", { name: "Sections" });
    expect(navLinks()).toEqual(["Overview"]);
  });

  it("the top bar shows the brand, the person and the role from /me", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: managerMe } });

    renderApp("/");

    const top = await screen.findByRole("banner");
    expect(top).toHaveTextContent("Chai Point Demo");
    expect(top).toHaveTextContent("Arjun Mehta, manager, Indiranagar");
  });

  it("Sign out calls logout and goes to /sign-in", async () => {
    const fetchStub = stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "POST /api/v1/auth/logout": { status: 204 },
    });
    renderApp("/");

    fireEvent.click(await screen.findByRole("button", { name: "Sign out" }));

    expect(
      await screen.findByRole("heading", { name: "Sign in to review intelligence" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/sign-in");
    expect(fetchStub.mock.calls.map((c) => `${c[1]?.method ?? "GET"} ${String(c[0])}`)).toContain(
      "POST /api/v1/auth/logout",
    );
  });

  it("Sign out still goes to /sign-in when logout fails", async () => {
    stubFetch({
      "GET /api/v1/me": { status: 200, body: adminMe },
      "POST /api/v1/auth/logout": errorReply(500, "internal", "Something failed."),
    });
    renderApp("/");

    fireEvent.click(await screen.findByRole("button", { name: "Sign out" }));

    expect(
      await screen.findByRole("heading", { name: "Sign in to review intelligence" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/sign-in");
  });
});
