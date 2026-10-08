import { useLoad } from "../../lib/use-load";
import { getTaggingStatus } from "./api";

// StatusStrip is the top bar's tagging and budget line, read once per page
// view. Colour always comes with the words.
export function StatusStrip() {
  const [state] = useLoad("status", getTaggingStatus);
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
