import { useState } from "react";
import { useNavigate } from "react-router";
import { useSession } from "../../app/session-context";
import { logout } from "./api";

// useSignOut is the one sign-out action, shared by the top bar and the
// phone's More page. Signed out either way: if the call fails the cookie
// expires on its own.
export function useSignOut(): [() => Promise<void>, boolean] {
  const { signedOut } = useSession();
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  async function signOut() {
    setBusy(true);
    await logout().catch(() => undefined);
    signedOut();
    navigate("/sign-in", { replace: true });
  }
  return [signOut, busy];
}
