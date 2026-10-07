// Who is signed in, for every screen. The components that provide and guard
// it live in session.tsx; this file holds no components, so fast refresh and
// react-refresh/only-export-components stay happy.
import { createContext, useContext } from "react";
import type { Me } from "../features/auth/api";
import type { ApiError } from "../lib/api";

// expired is true only when a session ended during use (a 401 after sign-in);
// a first visit with no cookie is signed-out with expired false.
export type Session =
  | { status: "loading" }
  | { status: "signed-out"; expired: boolean }
  | { status: "signed-in"; me: Me }
  | { status: "error"; error: ApiError };

export type SessionValue = {
  session: Session;
  signedIn: (me: Me) => void;
  signedOut: () => void;
};

export const SessionContext = createContext<SessionValue | null>(null);

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (value === null) {
    throw new Error("useSession is used outside SessionProvider");
  }
  return value;
}
