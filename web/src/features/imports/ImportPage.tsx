import { useEffect, useId, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { useSession } from "../../app/session-context";
import { asApiError, type ApiError } from "../../lib/api";
import { listOutlets } from "../outlets/api";
import { importReviews, type ImportResult } from "./api";
import { ImportResultCard } from "./ImportResultCard";

type Outlets = "loading" | "none" | "some" | "unknown";

type Failure = { message: string; requestId?: string };

// failureFor turns each refusal the spec lists for POST /imports into what
// happened and what to do; csv_invalid and file_too_large carry the
// server's own sentence, which names the column or the limit.
function failureFor(err: ApiError): Failure {
  switch (err.code) {
    case "csv_invalid":
    case "file_too_large":
      return { message: err.message };
    case "role_not_allowed":
      return { message: "Only the brand admin can import reviews." };
    case "network":
      return { message: "The file was not imported because the server did not answer. Try again." };
  }
  return {
    message: "The file was not imported because something failed on the server. Try again.",
    requestId: err.requestId,
  };
}

// ImportPage is S-07 for the brand admin: one file field, then the result.
export function ImportPage() {
  const { session } = useSession();
  const [outlets, setOutlets] = useState<Outlets>("loading");
  const [file, setFile] = useState<File | null>(null);
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<Failure | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const id = useId();
  const errorId = `${id}-error`;

  useEffect(() => {
    let live = true;
    listOutlets().then(
      (list) => live && setOutlets(list.length > 0 ? "some" : "none"),
      // The form still works; the server reports unknown outlets per row.
      () => live && setOutlets("unknown"),
    );
    return () => {
      live = false;
    };
  }, []);

  function choose(chosen: File | null) {
    setFile(chosen);
    setKey(chosen ? crypto.randomUUID() : "");
    setFailure(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    if (!file) {
      setFailure({ message: "Choose a CSV file to import." });
      return;
    }
    setBusy(true);
    setFailure(null);
    try {
      const res = await importReviews(file, key);
      setResult(res);
      setFile(null);
      setKey("");
      form.reset();
    } catch (err) {
      setFailure(failureFor(asApiError(err)));
    } finally {
      setBusy(false);
    }
  }

  const timezone = session.status === "signed-in" ? session.me.brand.timezone : "";

  return (
    <div className="content cols-2">
      <div className="span-all page-head">
        <h1>
          Import <span className="hl">reviews</span>
        </h1>
        <p className="muted">
          Upload a CSV export of reviews. Google reviews will connect here later; only CSV is
          available now.
        </p>
      </div>
      {busy && file && (
        <section aria-busy="true">
          <p role="status">Reading {file.name} and checking each row.</p>
        </section>
      )}
      {!busy && result && <ImportResultCard result={result} />}
      {!busy && !result && outlets === "none" && (
        <section className="empty">
          <h2>Add an outlet first</h2>
          <p>Each CSV row is matched to an outlet by name, so add your outlets before importing.</p>
          <Link className="btn btn-primary" to="/outlets">
            Add outlets
          </Link>
        </section>
      )}
      {!busy && !result && outlets !== "none" && (
        <section>
          <h2>Before you import</h2>
          <p>
            Export reviews as CSV with one row per review. Dates may be any day; reviews are grouped
            into Monday to Sunday weeks in the {timezone} time zone.
          </p>
        </section>
      )}
      <aside>
        <form className="add-outlet" onSubmit={submit}>
          <label htmlFor={`${id}-file`}>CSV file</label>
          <input
            id={`${id}-file`}
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => choose(e.target.files?.[0] ?? null)}
            aria-invalid={failure !== null}
            aria-describedby={`${id}-hint${failure ? ` ${errorId}` : ""}`}
          />
          <div className="error" id={errorId} role={failure ? "alert" : undefined}>
            {failure && <p>{failure.message}</p>}
            {failure?.requestId && <p className="request-id">Request id: {failure.requestId}</p>}
          </div>
          <p className="small muted" id={`${id}-hint`}>
            Columns needed: outlet, source, date, rating, text, reviewer_name. The outlet column
            must match an outlet name. Rows already imported are skipped, so you can import a
            corrected file again.
          </p>
          <button className="btn btn-primary" type="submit" disabled={busy || outlets === "none"}>
            {busy ? "Importing" : "Import reviews"}
          </button>
        </form>
      </aside>
    </div>
  );
}
