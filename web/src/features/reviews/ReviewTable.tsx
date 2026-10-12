import { Link } from "react-router";
import { formatDate, reasonLabels, sentimentBadges, sentimentLabels } from "../../lib/format";
import type { ReviewSummary, Theme } from "./api";
import { TableScroll } from "../../components/TableScroll";

const replyLabels: Record<string, string> = {
  none: "Not replied",
  drafting: "Drafting",
  draft: "Draft ready",
  replied: "Replied",
};

const replyBadges: Record<string, string> = {
  drafting: "neutral",
  draft: "neutral",
  replied: "success",
};

// ReviewTable lists reviews newest first. Review text and names are rendered
// as text (tenet 6); urgent reviews carry a red label with the word. Each
// cell names its column in data-label, which the phone layout shows when the
// rows stack into cards.
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
    <TableScroll label="Reviews">
      <table className="data reviews stack">
        <caption className="sr-only">Reviews, newest first</caption>
        <thead>
          <tr>
            <th scope="col">Date</th>
            {showOutlet && <th scope="col">Outlet</th>}
            <th scope="col">Rating</th>
            <th scope="col">Review</th>
            <th scope="col">Themes</th>
            <th scope="col">Sentiment</th>
            <th scope="col">Reply</th>
          </tr>
        </thead>
        <tbody>
          {reviews.map((r) => (
            <tr key={r.id}>
              <td className="nowrap" data-label="Date">
                <Link to={`/reviews/${r.id}`} aria-label={`Open review ${r.id}`}>
                  {formatDate(r.review_date)}
                </Link>
              </td>
              {showOutlet && <td data-label="Outlet">{r.outlet.name}</td>}
              <td data-label="Rating">
                <span className="rating" aria-label={`${r.rating} out of 5`}>
                  {r.rating}/5
                </span>
              </td>
              <td className="review-cell" data-label="Review">
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
              <td data-label="Themes">
                {r.tags === null ? (
                  <span className="muted">Not tagged yet</span>
                ) : r.tags.themes.length === 0 ? (
                  <span className="muted">None</span>
                ) : (
                  r.tags.themes.map((code) => (
                    <span key={code} className="chip">
                      {label(code)}
                    </span>
                  ))
                )}
              </td>
              <td data-label="Sentiment">
                {r.tags ? (
                  <span className={`badge ${sentimentBadges[r.tags.sentiment] ?? ""}`}>
                    {sentimentLabels[r.tags.sentiment]}
                  </span>
                ) : (
                  <span className="muted">Not tagged yet</span>
                )}
              </td>
              <td data-label="Reply">
                {replyBadges[r.reply_status] ? (
                  <span className={`badge ${replyBadges[r.reply_status]}`}>
                    {replyLabels[r.reply_status]}
                  </span>
                ) : (
                  <span className="muted">{replyLabels[r.reply_status]}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableScroll>
  );
}
