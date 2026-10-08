import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { adminMe, errorReply, stubFetch, unauthorized, type Reply } from "../../test/fetch-stub";
import { renderApp } from "../../test/render-app";
import { expectAccessibleStructure } from "../../test/a11y";

async function fillAndSubmit(email = "ritika.rao@example.in", password = "correct horse") {
  fireEvent.change(await screen.findByLabelText("Work email"), { target: { value: email } });
  fireEvent.change(screen.getByLabelText("Password"), { target: { value: password } });
  fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
}

function signedOutWithLogin(reply: Reply | (() => Promise<Reply>)) {
  return stubFetch({ "GET /api/v1/me": unauthorized, "POST /api/v1/auth/login": reply });
}

describe("SignInPage", () => {
  it("signing in navigates to the overview", async () => {
    const fetchStub = signedOutWithLogin({ status: 200, body: adminMe });
    renderApp("/sign-in");

    await fillAndSubmit();

    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/");
    const login = fetchStub.mock.calls.find((c) => String(c[0]) === "/api/v1/auth/login");
    expect(login?.[1]?.body).toBe('{"email":"ritika.rao@example.in","password":"correct horse"}');
  });

  it("the button reads Signing in and is disabled while the request runs", async () => {
    let answer: (r: Reply) => void = () => undefined;
    signedOutWithLogin(() => new Promise<Reply>((resolve) => (answer = resolve)));
    renderApp("/sign-in");

    await fillAndSubmit();

    const button = await screen.findByRole("button", { name: "Signing in" });
    expect(button).toBeDisabled();
    expect(screen.getByLabelText("Work email")).toHaveValue("ritika.rao@example.in");
    answer(errorReply(401, "invalid_credentials", "No match."));
    expect(await screen.findByRole("button", { name: "Sign in" })).toBeEnabled();
  });

  it("a wrong password marks both fields and shows the shared message", async () => {
    signedOutWithLogin(errorReply(401, "invalid_credentials", "No match."));
    renderApp("/sign-in");

    await fillAndSubmit();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "That email and password do not match an account. Check both and try again.",
    );
    expect(screen.getByLabelText("Work email")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Password")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByTestId("location").textContent).toBe("/sign-in");
    // each invalid field points at the message that explains it
    for (const name of ["Work email", "Password"]) {
      const id = screen.getByLabelText(name).getAttribute("aria-describedby") ?? "";
      expect(id).not.toBe("");
      expect(document.getElementById(id.split(" ")[0] ?? "")).not.toBeNull();
    }
    expectAccessibleStructure();
  });

  it("a 422 shows the message under the named field", async () => {
    signedOutWithLogin(
      errorReply(422, "validation_failed", "email is too long", [
        { field: "email", reason: "too_long" },
      ]),
    );
    renderApp("/sign-in");

    await fillAndSubmit();

    const email = screen.getByLabelText("Work email");
    expect(await screen.findByText("Use at most 200 characters.")).toBeInTheDocument();
    expect(email).toHaveAttribute("aria-invalid", "true");
    expect(email).toHaveAccessibleDescription("Use at most 200 characters.");
    expect(screen.getByLabelText("Password")).toHaveAttribute("aria-invalid", "false");
  });

  it("a server that does not answer gets the retry message with the request id", async () => {
    signedOutWithLogin(errorReply(500, "internal", "Something failed."));
    renderApp("/sign-in");

    await fillAndSubmit();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Sign-in failed because the server did not answer.");
    expect(alert).toHaveTextContent("Request id: req-test-1");
  });

  it("a 403 cross_site_request shows the server's message", async () => {
    signedOutWithLogin(
      errorReply(403, "cross_site_request", "Open OutletOwl from its own address."),
    );
    renderApp("/sign-in");

    await fillAndSubmit();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Open OutletOwl from its own address.",
    );
  });

  it("expired=1 shows the session ended notice", async () => {
    stubFetch({ "GET /api/v1/me": unauthorized });

    renderApp("/sign-in?expired=1");

    expect(await screen.findByRole("status")).toHaveTextContent(
      "Signed out. Your session ended after 8 hours. Sign in again to continue.",
    );
  });

  it("the heading names the product, not the brand", async () => {
    stubFetch({ "GET /api/v1/me": unauthorized });

    renderApp("/sign-in");

    const heading = await screen.findByRole("heading", { level: 1 });
    expect(heading).toHaveTextContent("Sign in to review intelligence");
    expect(heading).not.toHaveTextContent(adminMe.brand.name);
  });

  it("a signed-in visitor is sent to the overview", async () => {
    stubFetch({ "GET /api/v1/me": { status: 200, body: adminMe } });

    renderApp("/sign-in");

    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByTestId("location").textContent).toBe("/");
  });
});
