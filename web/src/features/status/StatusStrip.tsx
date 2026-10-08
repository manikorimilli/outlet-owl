import { useLocation } from "react-router";
import { useLoad } from "../../lib/use-load";
import { getTaggingStatus } from "./api";

// StatusStrip is the top bar's tagging and budget line. The shell stays
// mounted, so it is read again on every navigation. Colour always comes with
// the words.
export function StatusStrip() {
  const { key } = useLocation();
  const [state] = useLoad(`status-${key}`, getTaggingStatus);
  if (state.status === "error") {
    return <span className="badge status warn">Tagging status unavailable</span>;
  }
  if (state.status !== "ready") {
    return null;
  }
  const s = state.data;
  let text: string;
  let tone = "";
  if (s.budget.state !== "ok") {
    text = "Model budget used up";
    tone = "warn";
  } else if (s.worker === "paused") {
    text = `Tagging paused, ${s.untagged_count} waiting`;
    tone = "warn";
  } else if (s.untagged_count > 0) {
    text = `Tagging ${s.untagged_count} new ${s.untagged_count === 1 ? "review" : "reviews"}`;
  } else {
    text = "All reviews tagged";
  }
  return <span className={`badge status ${tone}`}>{text}</span>;
}
