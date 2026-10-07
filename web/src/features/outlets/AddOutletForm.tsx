import { useId, useState, type FormEvent } from "react";
import { asApiError, type ApiError } from "../../lib/api";
import { createOutlet, type OutletSummary } from "./api";

type Failure = { message: string; requestId?: string };

// failureFor turns each refusal the spec lists for POST /outlets into what
// happened and what to do (web LLD 4.4); a known code never reads as generic.
function failureFor(err: ApiError, name: string): Failure {
  switch (err.code) {
    case "outlet_name_taken": {
      const existing = err.details[0]?.reason ?? name.trim();
      return {
        message: `An outlet named ${existing} already exists. Names are matched without regard to capitals, so use a different name.`,
      };
    }
    case "validation_failed": {
      const reason = err.details.find((d) => d.field === "name")?.reason;
      if (reason === "blank") {
        return { message: "Enter the outlet name." };
      }
      if (reason === "too_long") {
        return { message: "Use at most 200 characters." };
      }
      return { message: err.message, requestId: err.requestId };
    }
    case "cross_site_request":
      return { message: err.message, requestId: err.requestId };
    case "role_not_allowed":
      return { message: "Only the brand admin can add outlets." };
  }
  return {
    message: "The outlet was not added because the server did not answer. Try again.",
    requestId: err.requestId,
  };
}

// AddOutletForm adds one outlet; the page refetches the list when it has.
export function AddOutletForm({ onAdded }: { onAdded: (outlet: OutletSummary) => void }) {
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<Failure | null>(null);
  const id = useId();
  const errorId = `${id}-error`;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(null);
    try {
      const outlet = await createOutlet(name);
      setName("");
      onAdded(outlet);
    } catch (err) {
      setFailure(failureFor(asApiError(err), name));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="add-outlet" onSubmit={submit}>
      <h2>Add an outlet</h2>
      <label htmlFor={`${id}-name`}>Outlet name</label>
      <input
        id={`${id}-name`}
        required
        maxLength={200}
        value={name}
        onChange={(e) => setName(e.target.value)}
        aria-invalid={failure !== null}
        aria-describedby={failure ? errorId : undefined}
      />
      <div className="error" id={errorId} role={failure ? "alert" : undefined}>
        {failure && <p>{failure.message}</p>}
        {failure?.requestId && <p className="request-id">Request id: {failure.requestId}</p>}
      </div>
      <p className="small muted">
        The name must match the outlet column in your CSV files. The outlet&apos;s manager is added
        in the users file.
      </p>
      <button className="btn btn-primary" type="submit" disabled={busy}>
        Add outlet
      </button>
    </form>
  );
}
