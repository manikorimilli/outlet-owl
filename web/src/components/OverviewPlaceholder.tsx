import { useSession } from "../app/session-context";

// The overview route until build phase 4 brings S-02.
export function OverviewPlaceholder() {
  const { session } = useSession();
  const outlet = session.status === "signed-in" ? session.me.outlet : null;
  return (
    <div className="content">
      <div className="page-head">
        <h1>Overview</h1>
        <p className="muted">The overview arrives with the dashboard.</p>
        {outlet && <p>Your outlet: {outlet.name}</p>}
      </div>
    </div>
  );
}
