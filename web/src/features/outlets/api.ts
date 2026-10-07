import { apiFetch } from "../../lib/api";
import type { components, operations } from "../../lib/api-types";

export type OutletSummary = components["schemas"]["OutletSummary"];
type OutletCreate = components["schemas"]["OutletCreate"];
type OutletList = operations["listOutlets"]["responses"][200]["content"]["application/json"];

// listOutlets returns the outlets the caller may see, in the server's order
// (name, ignoring capitals).
export async function listOutlets(): Promise<OutletSummary[]> {
  const list = await apiFetch<OutletList>("/outlets");
  return list.data;
}

export function createOutlet(name: string): Promise<OutletSummary> {
  const body: OutletCreate = { name };
  return apiFetch<OutletSummary>("/outlets", { method: "POST", body });
}
