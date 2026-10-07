import { useSession } from "../app/session-context";

// The overview route until build phase 4 brings S-02.
export function OverviewPlaceholder() {
  const { session } = useSession();
  const outlet = session.status === "signed-in" ? session.me.outlet : null;
  return (
    <section className="page" aria-labelledby="overview-heading">
      <h1 id="overview-heading">Overview</h1>
      <p>The overview arrives with the dashboard.</p>
      {outlet && <p>Your outlet: {outlet.name}</p>}
    </section>
  );
}
