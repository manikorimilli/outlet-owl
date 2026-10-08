import { Sparkline } from "../../components/Sparkline";
import { negativeShare, type Trends } from "./api";

function pct(v: number | null): string {
  return v === null ? "none" : `${Math.round(v * 100)}%`;
}

// OutletComparison is one ruled table: the latest week's rating and its
// change, and 12 week sparklines of rating and negative share (S-02).
export function OutletComparison({ trends }: { trends: Trends }) {
  return (
    <div className="table-scroll">
      <table className="data">
        <caption className="sr-only">
          Outlets compared, latest complete week and 12 week trends
        </caption>
        <thead>
          <tr>
            <th scope="col">Outlet</th>
            <th scope="col" className="num">
              Rating
            </th>
            <th scope="col">Change</th>
            <th scope="col">12 weeks</th>
            <th scope="col" className="num">
              Negative
            </th>
            <th scope="col">12 weeks</th>
          </tr>
        </thead>
        <tbody>
          {trends.outlets.map(({ outlet, weeks }) => {
            const last = weeks[weeks.length - 1];
            const prev = weeks[weeks.length - 2];
            const ratings = weeks.map((w) => w.average_rating);
            const shares = weeks.map(negativeShare);
            const firstRating = ratings.find((r) => r !== null) ?? null;
            const firstShare = shares.find((s) => s !== null) ?? null;
            const change =
              last?.average_rating != null && prev?.average_rating != null
                ? Math.round((last.average_rating - prev.average_rating) * 10) / 10
                : null;
            return (
              <tr key={outlet.id}>
                <th scope="row">{outlet.name}</th>
                <td className="num">{last?.average_rating ?? "none"}</td>
                <td>
                  {change === null || change === 0
                    ? "no change"
                    : `${change > 0 ? "up" : "down"} ${Math.abs(change)}`}
                </td>
                <td>
                  <Sparkline
                    values={ratings}
                    max={5}
                    label={`${outlet.name} rating over 12 weeks, ${firstRating ?? "none"} to ${last?.average_rating ?? "none"}`}
                  />
                </td>
                <td className="num">{pct(last ? negativeShare(last) : null)}</td>
                <td>
                  <Sparkline
                    values={shares}
                    max={1}
                    label={`${outlet.name} negative share over 12 weeks, ${pct(firstShare)} to ${pct(last ? negativeShare(last) : null)}`}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
