# Recordings

Saved model answers, one file per request: `<purpose>/<sha256 of the exact
request body>.json`. The gateway writes them in record mode
(`MODEL_GATEWAY_MODE=record`, a deliberate paid run) and answers from them
in replay mode, the default, which tests and CI use (tenet 5). A missing
recording fails with its key and path; nothing falls back to a live call.

Recordings hold seed text only, never real customer reviews (HLD section 4).
A new prompt version or a change to the request body makes the matching
recordings stale: record them again on purpose.
