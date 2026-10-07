import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, managerMe, stubFetch } from "../test/fetch-stub";
import { renderApp } from "../test/render-app";

describe("the routes", () => {
  it("RequireRole sends an outlet manager from /outlets to /", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: managerMe } });

    renderApp("/outlets");

    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/");
    expect(screen.queryByRole("heading", { name: "Outlets" })).not.toBeInTheDocument();
  });

  it("the brand admin reaches /outlets", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: adminMe } });

    renderApp("/outlets");

    expect(await screen.findByRole("heading", { name: "Outlets" })).toBeInTheDocument();
  });

  it("the overview names the manager's outlet", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: managerMe } });

    renderApp("/");

    expect(await screen.findByText("Your outlet: Indiranagar")).toBeInTheDocument();
  });

  it("an unknown path goes to /", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: adminMe } });

    renderApp("/no-such-page");

    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/");
  });
});
