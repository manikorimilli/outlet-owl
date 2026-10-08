import { useState } from "react";
import { useSession } from "../../app/session-context";
import { apiFetch, asApiError, type ApiError } from "../../lib/api";
import type { components } from "../../lib/api-types";
import { formatRange } from "../../lib/format";
import { useLoad } from "../../lib/use-load";
import { getMovers } from "../overview/api";

type Digest = components["schemas"]["Digest"];

// DigestPage is S-08 for the brand admin: one button that generates the
// weekly digest now and sends it to MailHog (Q-005, Q-006). Each press makes
// a new key; the button is off while a request runs, so a double click sends
// one email (tenet 8).
// newKey is a v4 UUID. crypto.randomUUID exists only in secure contexts, so
// the UI opened over plain http on a LAN address falls back to random bytes.
function newKey(): string {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  const b = crypto.getRandomValues(new Uint8Array(16));
  b[6] = ((b[6] ?? 0) & 0x0f) | 0x40;
  b[8] = ((b[8] ?? 0) & 0x3f) | 0x80;
  const h = [...b].map((x) => x.toString(16).padStart(2, "0")).join("");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

export function DigestPage() {
  const { session } = useSession();
  const email = session.status === "signed-in" ? session.me.email : "";
  const [movers] = useLoad("movers", getMovers);
  const [busy, setBusy] = useState(false);
  const [digest, setDigest] = useState<Digest | null>(null);
  const [failure, setFailure] = useState<ApiError | null>(null);

  async function generate() {
    setBusy(true);
    setFailure(null);
    try {
      setDigest(
        await apiFetch<Digest>("/digests", {
          method: "POST",
          headers: { "Idempotency-Key": newKey() },
        }),
      );
    } catch (err) {
      setFailure(asApiError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="content cols-2">
      <div className="span-all page-head">
        <h1>
          Weekly <span className="hl">digest</span>
        </h1>
        <p className="muted">
          {movers.status === "ready"
            ? `Covers the latest complete week, ${formatRange(movers.data.week.start, movers.data.week.end)}, compared with ${formatRange(movers.data.previous_week.start, movers.data.previous_week.end, false)}. Sent to ${email}.`
            : `Covers the latest complete Monday to Sunday week. Sent to ${email}.`}
        </p>
      </div>
      <section aria-labelledby="contains-title">
        <h2 id="contains-title">What it will contain</h2>
        <ul>
          <li>The outlet and theme that moved most, and the next four movers</li>
          <li>Every urgent review dated in the week</li>
          <li>A warning line if any review in the two weeks is not tagged</li>
        </ul>
      </section>
      <aside>
        <button className="btn btn-primary" type="button" disabled={busy} onClick={generate}>
          {busy ? "Sending" : "Generate and send digest"}
        </button>
        <p className="small muted">Sends one e-mail to {email} through MailHog.</p>
        <p role="status">{busy ? "Building the digest and sending it." : ""}</p>
        {failure && (
          <div className="error" role="alert">
            <p>
              {failure.code === "mail_unavailable"
                ? "The digest was not sent: the mail catcher (MailHog) did not answer. Start MailHog, then generate the digest again."
                : failure.message}
            </p>
            {failure.requestId && <p className="request-id">Request id: {failure.requestId}</p>}
          </div>
        )}
      </aside>
      {digest && (
        <section className="span-all" aria-labelledby="digest-title">
          <h2 id="digest-title">{digest.subject}</h2>
          <p className="badge success">
            {digest.status === "sent" ? "Sent" : "Sending"} to {digest.recipient_email}, week of{" "}
            {formatRange(digest.week_start, digest.week_end)}
          </p>
          <pre className="digest-body">{digest.body}</pre>
        </section>
      )}
    </div>
  );
}
