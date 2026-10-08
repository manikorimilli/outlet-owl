import { useId, useState, type FormEvent } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router";
import { useSession } from "../../app/session-context";
import { asApiError, type ApiError } from "../../lib/api";
import { login } from "./api";

// What the last attempt got wrong. A wrong password marks both fields and never
// says which was wrong, so accounts cannot be guessed (S-01 error state).
type Problem =
  | { kind: "none" }
  | { kind: "credentials" }
  | { kind: "fields"; email?: string; password?: string }
  | { kind: "form"; message: string; requestId?: string };

const credentialsMessage =
  "That email and password do not match an account. Check both and try again.";
const serverMessage =
  "Sign-in failed because the server did not answer. Check that OutletOwl is running, then try again.";

function fieldMessage(field: string, reason: string): string | undefined {
  switch (reason) {
    case "blank":
      return field === "email" ? "Enter your work email." : "Enter your password.";
    case "too_long":
      return "Use at most 200 characters.";
    case "invalid_email":
      return "Enter an email address, like name@example.in.";
  }
  return undefined;
}

function problemFor(err: ApiError): Problem {
  if (err.code === "invalid_credentials") {
    return { kind: "credentials" };
  }
  if (err.code === "validation_failed") {
    const fields: { email?: string; password?: string } = {};
    for (const d of err.details) {
      if (d.field === "email" || d.field === "password") {
        fields[d.field] ??= fieldMessage(d.field, d.reason) ?? err.message;
      }
    }
    if (fields.email || fields.password) {
      return { kind: "fields", ...fields };
    }
    return { kind: "form", message: err.message, requestId: err.requestId };
  }
  if (err.code === "cross_site_request") {
    return { kind: "form", message: err.message, requestId: err.requestId };
  }
  return { kind: "form", message: serverMessage, requestId: err.requestId };
}

// SignInPage is S-01: the brand is not named here (it is shown in the top bar
// after sign-in, from GET /me), a decision recorded in the web LLD.
// demoAccount fills the form with the seeded brand admin under the Vite dev
// server only (make web-dev). Tests run in "test" mode and builds in
// "production", so no bundle the Go binary serves ever carries it.
const demoAccount =
  import.meta.env.MODE === "development"
    ? { email: "ritika.rao@example.in", password: "outletowl-demo" }
    : null;

export function SignInPage() {
  const { session, signedIn } = useSession();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [email, setEmail] = useState(demoAccount?.email ?? "");
  const [password, setPassword] = useState(demoAccount?.password ?? "");
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState<Problem>({ kind: "none" });
  const id = useId();

  if (session.status === "signed-in") {
    return <Navigate to="/" replace />;
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setProblem({ kind: "none" });
    try {
      const me = await login({ email, password });
      signedIn(me);
      navigate("/", { replace: true });
    } catch (err) {
      setProblem(problemFor(asApiError(err)));
      setBusy(false);
    }
  }

  const errorId = `${id}-error`;
  const emailErrorId = `${id}-email-error`;
  const passwordErrorId = `${id}-password-error`;
  const both = problem.kind === "credentials";
  const emailError = problem.kind === "fields" ? problem.email : undefined;
  const passwordError = problem.kind === "fields" ? problem.password : undefined;
  const shared =
    problem.kind === "credentials"
      ? credentialsMessage
      : problem.kind === "form"
        ? problem.message
        : undefined;

  return (
    <main id="main" className="signin-page">
      <div className="signin-wrap">
        <div className="signin-intro">
          <p className="brand big">
            <svg className="logo" viewBox="0 0 32 32" aria-hidden="true">
              <rect className="logo-b" x="2" y="3" width="28" height="22" rx="9" />
              <path className="logo-b" d="M9 23l-1.5 7 8-7z" />
              <path
                className="logo-s"
                d="M16 8.5l2 4.1 4.5.6-3.3 3.1.8 4.4-4-2.1-4 2.1.8-4.4-3.3-3.1 4.5-.6z"
              />
            </svg>
            <span>OutletOwl</span>
          </p>
          <h1>
            Sign in to <span className="hl">review intelligence</span>
          </h1>
          <p className="muted">Movers, urgent reviews and reply drafts for every outlet.</p>
        </div>
        <form className="signin" onSubmit={submit}>
          {params.get("expired") === "1" && (
            <div className="notice warn" role="status">
              <p>
                <strong>Signed out.</strong> Your session ended after 8 hours. Sign in again to
                continue.
              </p>
            </div>
          )}
          <div>
            <label htmlFor={`${id}-email`}>Work email</label>
            <input
              id={`${id}-email`}
              type="email"
              autoComplete="username"
              required
              maxLength={200}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-invalid={both || emailError !== undefined}
              aria-describedby={both ? errorId : emailError ? emailErrorId : undefined}
            />
            {emailError && (
              <p className="error" id={emailErrorId}>
                {emailError}
              </p>
            )}
          </div>
          <div>
            <label htmlFor={`${id}-password`}>Password</label>
            <input
              id={`${id}-password`}
              type="password"
              autoComplete="current-password"
              required
              maxLength={200}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={both || passwordError !== undefined}
              aria-describedby={both ? errorId : passwordError ? passwordErrorId : undefined}
            />
            {passwordError && (
              <p className="error" id={passwordErrorId}>
                {passwordError}
              </p>
            )}
          </div>
          <div className="error" id={errorId} role={shared ? "alert" : undefined}>
            {shared && <p>{shared}</p>}
            {problem.kind === "form" && problem.requestId && (
              <p className="request-id">Request id: {problem.requestId}</p>
            )}
          </div>
          <button className="btn btn-primary btn-block" type="submit" disabled={busy}>
            {busy ? "Signing in" : "Sign in"}
          </button>
          <p className="muted small">
            Accounts are set up by your administrator. To get access or reset a password, ask them.
          </p>
        </form>
      </div>
    </main>
  );
}
