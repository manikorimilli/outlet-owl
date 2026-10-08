import { useState } from "react";
import { apiFetch, asApiError, type ApiError } from "../../lib/api";
import type { components } from "../../lib/api-types";
import { formatRange } from "../../lib/format";

type Digest = components["schemas"]["Digest"];

// DigestPage is S-08 for the brand admin: one button that generates the
// weekly digest now and sends it to MailHog (Q-005, Q-006). Each press makes
// a new key; the button is off while a request runs, so a double click sends
// one email (tenet 8).
export function DigestPage() {
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
          headers: { "Idempotency-Key": crypto.randomUUID() },
        }),
      );
    } catch (err) {
      setFailure(asApiError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="content">
      <div className="page-head">
        <h1>
          Weekly <span className="hl">digest</span>
        </h1>
        <p className="muted">
          Generates the digest for the latest complete Monday to Sunday week and emails it to you
          through the local mail catcher. Nothing is sent on a schedule.
        </p>
      </div>
      <section>
        <button className="btn btn-primary" type="button" disabled={busy} onClick={generate}>
          {busy ? "Generating" : "Generate digest"}
        </button>
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
      </section>
      {digest && (
        <section aria-labelledby="digest-title">
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
