import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type TaggingStatus = components["schemas"]["TaggingStatus"];

export function getTaggingStatus(): Promise<TaggingStatus> {
  return apiFetch<TaggingStatus>("/tagging/status");
}

// dollars shows cents as "8.03".
export function dollars(cents: number): string {
  return (cents / 100).toFixed(2);
}
