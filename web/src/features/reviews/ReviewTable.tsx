import { Link } from "react-router";
import { formatDate, reasonLabels, sentimentLabels } from "../../lib/format";
import type { ReviewSummary, Theme } from "./api";

const replyLabels: Record<string, string> = {
  none: "Not replied",
  drafting: "Drafting",
  draft: "Draft ready",
  replied: "Replied",
};

// ReviewTable lists reviews newest first. Review text and names are rendered
// as text (tenet 6); urgent reviews carry a red label with the word.
export function ReviewTable({
  reviews,
  themes,
  showOutlet,
}: {
  reviews: ReviewSummary[];
  themes: Theme[];
  showOutlet: boolean;
}) {
  const label = (code: string) => themes.find((t) => t.code === code)?.label ?? code;
  return (
    <div className="table-scroll">
      <table className="data reviews">
        <caption className="sr-only">Reviews, newest first</caption>
        <thead>
          <tr>
            <th scope="col">Date</th>
            {showOutlet && <th scope="col">Outlet</th>}
            <th scope="col" className="num">
              Rating
            </th>
            <th scope="col">Review</th>
            <th scope="col">Themes</th>
            <th scope="col">Sentiment</th>
            <th scope="col">Reply</th>
          </tr>
        </thead>
        <tbody>
          {reviews.map((r) => (
            <tr key={r.id}>
              <td className="nowrap">
                <Link to={`/reviews/${r.id}`} aria-label={`Open review ${r.id}`}>
                  {formatDate(r.review_date)}
                </Link>
              </td>
              {showOutlet && <td>{r.outlet.name}</td>}
              <td className="num">{r.rating}/5</td>
              <td className="review-cell">
                {r.tags?.urgent_reasons.map((reason) => (
                  <span key={reason} className="badge danger">
                    Urgent: {reasonLabels[reason] ?? reason}
                  </span>
                ))}
                <p className="review-text">{r.review_text}</p>
                <p className="small muted">
                  {r.reviewer_name}, {r.source}
                </p>
              </td>
              <td>
                {r.tags === null ? (
                  <span className="muted">Not tagged yet</span>
                ) : r.tags.themes.length === 0 ? (
                  <span className="muted">None</span>
                ) : (
                  r.tags.themes.map(label).join(", ")
                )}
              </td>
              <td>
                {r.tags ? (
                  sentimentLabels[r.tags.sentiment]
                ) : (
                  <span className="muted">Not tagged yet</span>
                )}
              </td>
              <td>{replyLabels[r.reply_status]}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
