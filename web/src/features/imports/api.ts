import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type ImportResult = components["schemas"]["ImportResult"];

// importReviews uploads one CSV file. key is made once per chosen file, so a
// second press after a lost answer returns the first result (tenet 8).
export function importReviews(file: File, key: string): Promise<ImportResult> {
  const form = new FormData();
  form.append("file", file, file.name);
  return apiFetch<ImportResult>("/imports", {
    method: "POST",
    form,
    headers: { "Idempotency-Key": key },
  });
}
