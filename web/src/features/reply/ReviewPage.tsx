import { useEffect, useId, useState } from "react";
import { Link, useParams } from "react-router";
import { asApiError, type ApiError } from "../../lib/api";
import { formatDate, reasonLabels, sentimentLabels } from "../../lib/format";
import { useLoad } from "../../lib/use-load";
import {
  draftReply,
  getReview,
  markReplied,
  saveReply,
  type Reply,
  type ReviewDetail,
} from "./api";

// ReviewRoute keys the page by id, so a draft typed for one review never
// shows on another (React reuses the element when only :id changes).
export function ReviewRoute() {
  const { id } = useParams();
  const n = Number(id);
  if (!Number.isInteger(n) || n < 1) {
    return (
      <div className="content">
        <p className="error" role="alert">
          That link does not name a review. <Link to="/reviews">Back to reviews</Link>
        </p>
      </div>
    );
  }
  return <ReviewPage key={n} id={n} />;
}

type Drafting = "idle" | "drafting" | "unavailable";

// ReviewPage is S-05: the review, then its reply. A manager of the outlet
// gets a draft on open (Q-007), edits it and marks it replied, which is the
// approval (Q-008). Everyone else reads.
function ReviewPage({ id }: { id: number }) {
  const [state, reload] = useLoad(`review-${id}`, () => getReview(id));
  if (state.status === "loading") {
    return (
      <div className="content">
        <p role="status">Loading the review</p>
      </div>
    );
  }
  if (state.status === "error") {
    return (
      <div className="content">
        <div className="error" role="alert">
          <p>
            {state.error.code === "not_found"
              ? "There is no such review among the ones you can see."
              : "The review could not load because the server did not answer. Check that OutletOwl is running, then reload."}
          </p>
        </div>
        <Link to="/reviews">Back to reviews</Link>
      </div>
    );
  }
  return <Loaded detail={state.data} reload={reload} />;
}

function Loaded({ detail, reload }: { detail: ReviewDetail; reload: () => void }) {
  const r = detail;
  return (
    <div className="content cols-2">
      <div className="span-all page-head">
        <p>
          <Link to="/reviews">Back to reviews</Link>
        </p>
        <h1>
          Review at <span className="hl">{r.outlet.name}</span>
        </h1>
        <p className="muted">
          {formatDate(r.review_date)}, {r.rating}/5, {r.source}
        </p>
      </div>
      <section aria-labelledby="review-title">
        <h2 id="review-title">{r.reviewer_name} wrote</h2>
        {r.tags?.urgent_reasons.map((reason) => (
          <span key={reason} className="badge danger">
            Urgent: {reasonLabels[reason] ?? reason}
          </span>
        ))}
        <p className="review-text">{r.review_text}</p>
        <p className="small muted">
          {r.tags
            ? `${sentimentLabels[r.tags.sentiment]}. Themes: ${r.tags.themes.length ? r.tags.themes.join(", ") : "none"}.`
            : "Not tagged yet."}
        </p>
      </section>
      <aside aria-labelledby="reply-title">
        <h2 id="reply-title">Reply</h2>
        {r.can_reply ? (
          <ReplyEditor id={r.id} initial={r.reply} onConflict={reload} />
        ) : (
          <ReadOnlyReply detail={r} />
        )}
      </aside>
    </div>
  );
}

function ReadOnlyReply({ detail }: { detail: ReviewDetail }) {
  const reply = detail.reply;
  const who = detail.outlet_managers.map((m) => m.name).join(", ");
  return (
    <>
      {reply?.status === "replied" && <RepliedNote reply={reply} />}
      {reply?.reply_text ? (
        <p className="review-text">{reply.reply_text}</p>
      ) : (
        <p className="muted">No reply yet.</p>
      )}
      <p className="small muted">
        {who
          ? `Only ${who}, the outlet's manager, can draft and mark the reply.`
          : "Nobody manages this outlet yet: add a manager to the users file."}
      </p>
    </>
  );
}

function RepliedNote({ reply }: { reply: Reply }) {
  return (
    <p className="badge success">
      Replied
      {reply.replied_by ? ` by ${reply.replied_by.name}` : ""}
      {reply.replied_at ? ` on ${formatDate(reply.replied_at.slice(0, 10))}` : ""}
    </p>
  );
}

function ReplyEditor({
  id,
  initial,
  onConflict,
}: {
  id: number;
  initial: Reply | null;
  onConflict: () => void;
}) {
  const fieldId = useId();
  const [reply, setReply] = useState<Reply | null>(initial);
  const [text, setText] = useState(initial?.reply_text ?? "");
  const [drafting, setDrafting] = useState<Drafting>(
    initial === null || initial.status === "drafting" ? "drafting" : "idle",
  );
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<ApiError | null>(null);
  const [saved, setSaved] = useState("");

  // Ask for the draft on open; a 202 (another tab is drafting) is asked
  // again after its Retry-After of 2 seconds, at most 25 times.
  useEffect(() => {
    if (drafting !== "drafting") {
      return;
    }
    let live = true;
    let tries = 0;
    const ask = () => {
      tries += 1;
      draftReply(id).then(
        (r) => {
          if (!live) return;
          if (r.status === "drafting" && tries < 25) {
            setTimeout(ask, 2000);
            return;
          }
          setReply(r);
          setText(r.reply_text ?? "");
          setDrafting(r.status === "drafting" ? "unavailable" : "idle");
        },
        (err: unknown) => {
          if (!live) return;
          setFailure(asApiError(err));
          setDrafting("unavailable");
        },
      );
    };
    ask();
    return () => {
      live = false;
    };
  }, [id, drafting]);

  async function act(kind: "save" | "replied") {
    setBusy(true);
    setFailure(null);
    setSaved("");
    try {
      let base = reply;
      if (kind === "replied" && base === null) {
        base = await saveReply(id, text, null); // a hand-written reply is stored first
      }
      const next =
        kind === "save"
          ? await saveReply(id, text, base?.updated_at ?? null)
          : await markReplied(id, text, base?.updated_at ?? null);
      setReply(next);
      setText(next.reply_text ?? text);
      setSaved(kind === "save" ? "Saved." : "Marked replied. Post this text on the review site.");
    } catch (err) {
      setFailure(asApiError(err));
    } finally {
      setBusy(false);
    }
  }

  if (drafting === "drafting") {
    return (
      <p role="status">Drafting a reply in the brand&apos;s tone. This takes a few seconds.</p>
    );
  }
  if (reply?.status === "replied") {
    return (
      <>
        <RepliedNote reply={reply} />
        <p className="review-text">{reply.reply_text}</p>
        <p role="status">{saved}</p>
      </>
    );
  }
  const unavailable = failure?.code === "budget_exhausted" || failure?.code === "model_unavailable";
  const conflict =
    failure?.code === "reply_changed" ||
    failure?.code === "draft_in_progress" ||
    failure?.code === "already_replied";
  return (
    <form className="reply-form" onSubmit={(e) => e.preventDefault()}>
      <div className="actions">
        <button
          className="btn btn-primary"
          type="button"
          disabled={busy || text.trim() === ""}
          onClick={() => act("replied")}
        >
          Mark replied
        </button>
        <button
          className="btn btn-secondary"
          type="button"
          disabled={busy || text.trim() === ""}
          onClick={() => act("save")}
        >
          Save
        </button>
      </div>
      {unavailable && (
        <p className="notice warn" role="alert">
          {failure?.message} You can write the reply yourself below.
        </p>
      )}
      {conflict && (
        <div className="error" role="alert">
          <p>{failure?.message}</p>
          <button className="btn btn-secondary btn-small" type="button" onClick={onConflict}>
            Reload the reply
          </button>
        </div>
      )}
      {failure && !unavailable && !conflict && (
        <p className="error" role="alert">
          {failure.message}
          {failure.requestId ? ` Request id: ${failure.requestId}` : ""}
        </p>
      )}
      <label htmlFor={fieldId}>
        {reply?.draft_text ? "Drafted reply, edit before marking replied" : "Your reply"}
      </label>
      <textarea
        id={fieldId}
        rows={10}
        maxLength={5000}
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <p className="small muted">
        Marking replied approves this text. Post it on the review site yourself; OutletOwl does not
        post replies.
      </p>
      <p role="status">{saved}</p>
    </form>
  );
}
