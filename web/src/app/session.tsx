import { useEffect, useState, type ReactNode } from "react";
import { Navigate, Outlet } from "react-router";
import { getMe, type Me } from "../features/auth/api";
import { ApiError, asApiError, setUnauthorizedHandler } from "../lib/api";
import { SessionContext, useSession, type Session } from "./session-context";

// SessionProvider asks the server who is signed in, once on load (web LLD
// 4.1), and ends the session when any later call answers 401 unauthorized
// (4.3). The cookie is HttpOnly, so GET /me is the only way to know.
export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session>({ status: "loading" });

  useEffect(() => {
    let current = true;
    getMe().then(
      (me) => {
        if (current) {
          setSession({ status: "signed-in", me });
        }
      },
      (err: unknown) => {
        if (!current) {
          return;
        }
        const apiErr = asApiError(err);
        setSession(
          apiErr.status === 401
            ? { status: "signed-out", expired: false }
            : { status: "error", error: apiErr },
        );
      },
    );
    return () => {
      current = false;
    };
  }, []);

  useEffect(() => {
    // Only a signed-in session can expire: the 401 that GET /me answers on a
    // first visit leaves the status to the load above.
    setUnauthorizedHandler(() =>
      setSession((s) => (s.status === "signed-in" ? { status: "signed-out", expired: true } : s)),
    );
    return () => setUnauthorizedHandler(undefined);
  }, []);

  const value = {
    session,
    signedIn: (me: Me) => setSession({ status: "signed-in", me }),
    signedOut: () => setSession({ status: "signed-out", expired: false }),
  };
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

// RequireSession renders its child routes for a signed-in user and sends
// everyone else to sign-in, with the expired notice only when a session ended.
export function RequireSession() {
  const { session } = useSession();
  switch (session.status) {
    case "loading":
      return (
        <p role="status" className="page-status">
          Checking your session
        </p>
      );
    case "error":
      return <ServerUnavailable error={session.error} />;
    case "signed-out":
      return <Navigate to={session.expired ? "/sign-in?expired=1" : "/sign-in"} replace />;
    case "signed-in":
      return <Outlet />;
  }
}

// RequireRole sends a user without the role to the overview; it sits inside
// RequireSession, so the session is signed in.
export function RequireRole({ role }: { role: Me["role"] }) {
  const { session } = useSession();
  if (session.status !== "signed-in" || session.me.role !== role) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}

function ServerUnavailable({ error }: { error: ApiError }) {
  return (
    <main className="page-error" role="alert">
      <h1>OutletOwl did not answer</h1>
      <p>OutletOwl did not answer. Check that it is running, then reload.</p>
      {error.requestId && <p className="request-id">Request id: {error.requestId}</p>}
      <button type="button" onClick={() => window.location.reload()}>
        Reload
      </button>
    </main>
  );
}
