import type { ImportResult } from "./api";

// shownRows caps the rejected-rows table; a file with a wrong date format
// can reject every row, and thousands of rows would stall the page.
const shownRows = 200;

// ImportResultCard answers the three counts the admin needs and lists each
// rejected row with its number and reason (S-07 success and partial states).
// Reasons quote the admin's own cells and are rendered as text (tenet 6).
export function ImportResultCard({ result }: { result: ImportResult }) {
  return (
    <section aria-labelledby="import-result-title">
      <h2 id="import-result-title">{result.file_name} imported</h2>
      <dl className="facts">
        <div>
          <dt>Imported</dt>
          <dd className="big num">{result.imported_count}</dd>
        </div>
        <div>
          <dt>Already imported, skipped</dt>
          <dd className="big num">{result.duplicate_count}</dd>
        </div>
        <div>
          <dt>Rejected</dt>
          <dd className="big num">{result.rejected_count}</dd>
        </div>
      </dl>
      {result.imported_count > 0 && (
        <p className="notice info">
          <strong>
            Tagging {result.imported_count} new {result.imported_count === 1 ? "review" : "reviews"}
            .
          </strong>{" "}
          Tagging has started on its own; the Outlets page shows how many are not tagged yet.
        </p>
      )}
      {result.rejections.length > 0 && (
        <>
          <h3>Rejected rows</h3>
          <div className="table-scroll">
            <table className="data">
              <caption className="sr-only">Rejected rows</caption>
              <thead>
                <tr>
                  <th scope="col" className="num">
                    Row
                  </th>
                  <th scope="col">Reason</th>
                </tr>
              </thead>
              <tbody>
                {result.rejections.slice(0, shownRows).map((r) => (
                  <tr key={r.row_number}>
                    <td className="num">{r.row_number}</td>
                    <td>{r.reason}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {result.rejections.length > shownRows && (
            <p className="small muted">
              Showing the first {shownRows} of {result.rejections.length} rejected rows.
            </p>
          )}
          <p className="small muted">
            Fix these rows in the file and import it again; rows already imported are skipped.
          </p>
        </>
      )}
    </section>
  );
}
