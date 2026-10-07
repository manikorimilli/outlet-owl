import type { OutletSummary } from "./api";
import { outletRowId } from "./row-id";

// OutletTable lists who manages each outlet. Names are rendered as text
// (tenet 6). An outlet without a manager says so in words: nobody can reply
// to its reviews until the users file names one.
export function OutletTable({ outlets }: { outlets: OutletSummary[] }) {
  return (
    <div className="table-scroll">
      <table className="data">
        <caption className="sr-only">Outlets</caption>
        <thead>
          <tr>
            <th scope="col">Outlet</th>
            <th scope="col">Manager</th>
            <th scope="col" className="num">
              Reviews
            </th>
            <th scope="col" className="num">
              Not tagged
            </th>
          </tr>
        </thead>
        <tbody>
          {outlets.map((o) => (
            <tr key={o.id}>
              <th scope="row" id={outletRowId(o.id)} tabIndex={-1}>
                {o.name}
              </th>
              <td>
                {o.managers.length > 0 ? (
                  o.managers.map((m) => m.name).join(", ")
                ) : (
                  <span className="badge warn">No manager: add one to the users file</span>
                )}
              </td>
              <td className="num">{o.review_count}</td>
              <td className="num">{o.untagged_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
