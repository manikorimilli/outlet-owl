import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, stubFetch, type Reply } from "../../test/fetch-stub";
import { renderApp } from "../../test/render-app";

async function submitWith(reply: Reply, name: string) {
  stubFetch({
    "GET /api/v1/me": { status: 200, body: adminMe },
    "GET /api/v1/outlets": { status: 200, body: { data: [] } },
    "POST /api/v1/outlets": reply,
  });
  renderApp("/outlets");
  fireEvent.change(await screen.findByLabelText("Outlet name"), { target: { value: name } });
  fireEvent.click(screen.getByRole("button", { name: "Add outlet" }));
  return screen.findByRole("alert");
}

describe("AddOutletForm", () => {
  it("a 409 shows the existing name from details", async () => {
    const alert = await submitWith(
      errorReply(409, "outlet_name_taken", "An outlet with this name exists.", [
        { field: "name", reason: "Koramangala" },
      ]),
      "koramangala",
    );

    expect(alert).toHaveTextContent(
      "An outlet named Koramangala already exists. Names are matched without regard to capitals, so use a different name.",
    );
    expect(screen.getByLabelText("Outlet name")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Outlet name")).toHaveValue("koramangala");
  });

  it("a 403 cross_site_request shows the server's message", async () => {
    const alert = await submitWith(
      errorReply(403, "cross_site_request", "Open OutletOwl from its own address."),
      "Whitefield",
    );

    expect(alert).toHaveTextContent("Open OutletOwl from its own address.");
  });

  it("a 422 too_long shows the length message", async () => {
    const alert = await submitWith(
      errorReply(422, "validation_failed", "name is too long", [
        { field: "name", reason: "too_long" },
      ]),
      "Whitefield",
    );

    expect(alert).toHaveTextContent("Use at most 200 characters.");
  });

  it("a 422 blank asks for the outlet name", async () => {
    const alert = await submitWith(
      errorReply(422, "validation_failed", "name is blank", [{ field: "name", reason: "blank" }]),
      "   ",
    );

    expect(alert).toHaveTextContent("Enter the outlet name.");
  });

  it("a server that does not answer gets the retry message with the request id", async () => {
    const alert = await submitWith(errorReply(500, "internal", "Something failed."), "Whitefield");

    expect(alert).toHaveTextContent(
      "The outlet was not added because the server did not answer. Try again.",
    );
    expect(alert).toHaveTextContent("Request id: req-test-1");
    expect(screen.getByRole("button", { name: "Add outlet" })).toBeEnabled();
  });
});
