import {
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from "react";
import { asApiError, type ApiError } from "../../lib/api";
import { AddOutletForm } from "./AddOutletForm";
import { listOutlets, type OutletSummary } from "./api";
import { OutletTable } from "./OutletTable";
import { outletRowId } from "./row-id";

type Load =
  | { status: "loading" }
  | { status: "error"; error: ApiError }
  | { status: "ready"; outlets: OutletSummary[] };

// load starts a request and applies its answer only if no newer request has
// started since, so an older list never overwrites a newer one (web LLD 5).
function load(latest: RefObject<number>, setLoad: Dispatch<SetStateAction<Load>>) {
  latest.current += 1;
  const n = latest.current;
  listOutlets().then(
    (outlets) => {
      if (n === latest.current) {
        setLoad({ status: "ready", outlets });
      }
    },
    (err: unknown) => {
      if (n === latest.current) {
        setLoad({ status: "error", error: asApiError(err) });
      }
    },
  );
}

// OutletsPage is S-06 for the brand admin: the outlets with their managers,
// and the add form, which is usable while the list loads.
export function OutletsPage() {
  const [state, setLoad] = useState<Load>({ status: "loading" });
  const latest = useRef(0);
  const focusAfterLoad = useRef<number | null>(null);

  useEffect(() => {
    load(latest, setLoad);
    return () => {
      latest.current += 1; // drop any answer that arrives after unmount
    };
  }, []);

  useEffect(() => {
    const id = focusAfterLoad.current;
    if (state.status !== "ready" || id === null) {
      return;
    }
    const row = document.getElementById(outletRowId(id));
    if (row) {
      focusAfterLoad.current = null;
      row.focus();
    }
  }, [state]);

  function reload() {
    setLoad({ status: "loading" });
    load(latest, setLoad);
  }

  function added(outlet: OutletSummary) {
    focusAfterLoad.current = outlet.id;
    load(latest, setLoad); // the table keeps showing until the new list arrives
  }

  return (
    <div className="content cols-2">
      <div className="span-all page-head">
        <h1>
          <span className="hl">Outlets</span>
        </h1>
        {state.status === "loading" && <p className="muted">Loading outlets</p>}
        {state.status === "ready" && (
          <p className="muted">
            <span className="num">{state.outlets.length}</span>{" "}
            {state.outlets.length === 1 ? "outlet." : "outlets."}
          </p>
        )}
      </div>
      {state.status === "loading" && (
        <section aria-busy="true">
          <span className="skeleton list-skeleton" aria-hidden="true" />
        </section>
      )}
      {state.status === "error" && (
        <section>
          <div className="error" role="alert">
            <p>
              Outlets could not load because the server did not answer. Check that OutletOwl is
              running, then reload.
            </p>
            {state.error.requestId && (
              <p className="request-id">Request id: {state.error.requestId}</p>
            )}
          </div>
          <button className="btn btn-secondary" type="button" onClick={reload}>
            Reload outlets
          </button>
        </section>
      )}
      {state.status === "ready" && state.outlets.length === 0 && (
        <section className="empty">
          <h2>No outlets yet</h2>
          <p>Add each outlet once, using the same name your CSV files use. Then import reviews.</p>
        </section>
      )}
      {state.status === "ready" && state.outlets.length > 0 && (
        <section>
          <OutletTable outlets={state.outlets} />
        </section>
      )}
      <aside>
        <AddOutletForm onAdded={added} />
      </aside>
    </div>
  );
}
