import { Link } from "react-router";
import { useSession } from "../../app/session-context";
import { useSignOut } from "../auth/use-sign-out";

// MorePage is S-09: on a phone the bottom bar has room for five tabs, so the
// brand admin's less frequent destinations and Sign out live here. On wider
// screens they are in the side nav and the More tab is hidden.
export function MorePage() {
  const { session } = useSession();
  const [signOut, busy] = useSignOut();
  const me = session.status === "signed-in" ? session.me : null;
  return (
    <div className="content">
      <div className="page-head">
        <h1>More</h1>
      </div>
      <ul className="more-list">
        <li>
          <Link to="/outlets">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M3 9l2-5h14l2 5M4 9v11h16V9M9 20v-6h6v6" />
            </svg>
            Outlets
          </Link>
        </li>
        <li>
          <Link to="/digest">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M3 5h18v14H3zM3 6l9 7 9-7" />
            </svg>
            Weekly digest
          </Link>
        </li>
        <li>
          <button type="button" onClick={signOut} disabled={busy}>
            Sign out
          </button>
        </li>
      </ul>
      {me && <p className="small muted">Signed in as {me.name}, brand admin.</p>}
    </div>
  );
}
