import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type ReviewDetail = components["schemas"]["ReviewDetail"];
export type Reply = components["schemas"]["Reply"];

export const getReview = (id: number) => apiFetch<ReviewDetail>(`/reviews/${id}`);

// inFlight holds the draft request running for each review. A second ask
// while one runs shares it: React's development mode mounts the page twice,
// and two requests would take turns holding the drafting claim, so a
// hand-written save after "drafting unavailable" met the other's claim (409).
const inFlight = new Map<number, Promise<Reply>>();

// draftReply returns the reply; status "drafting" (202) means another tab
// holds the claim and the caller asks again after a pause.
export function draftReply(id: number): Promise<Reply> {
  const running = inFlight.get(id);
  if (running) return running;
  const request = apiFetch<Reply>(`/reviews/${id}/draft`, { method: "POST" }).finally(() =>
    inFlight.delete(id),
  );
  inFlight.set(id, request);
  return request;
}

export function saveReply(id: number, text: string, basedOn: string | null): Promise<Reply> {
  return apiFetch<Reply>(`/reviews/${id}/reply`, {
    method: "PUT",
    body: { reply_text: text, based_on_updated_at: basedOn },
  });
}

export function markReplied(id: number, text: string, basedOn: string | null): Promise<Reply> {
  return apiFetch<Reply>(`/reviews/${id}/replied`, {
    method: "POST",
    body: { reply_text: text, based_on_updated_at: basedOn },
  });
}
