import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type ReviewDetail = components["schemas"]["ReviewDetail"];
export type Reply = components["schemas"]["Reply"];

export const getReview = (id: number) => apiFetch<ReviewDetail>(`/reviews/${id}`);

// draftReply returns the reply, or null while another request drafts it
// (202): the caller asks again after a pause.
export async function draftReply(id: number): Promise<Reply> {
  return apiFetch<Reply>(`/reviews/${id}/draft`, { method: "POST" });
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
