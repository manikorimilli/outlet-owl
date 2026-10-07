// The one way the UI reaches the server (phase 1 web LLD, section 5):
// same-origin paths under /api/v1, the session cookie sent by the browser, and
// every answer that is not 2xx turned into an ApiError.
import type { components } from "./api-types";

type ErrorEnvelope = components["schemas"]["Error"];
export type ErrorDetail = NonNullable<ErrorEnvelope["error"]["details"]>[number];

const basePath = "/api/v1";

// ApiError is every failed call. code is the envelope's code, "network" when
// fetch itself failed, or "internal" when the body is not the envelope.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: ErrorDetail[];
  readonly requestId: string | undefined;

  constructor(
    status: number,
    code: string,
    message: string,
    details: ErrorDetail[] = [],
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }
}

// asApiError gives a promise rejection its type: apiFetch only throws ApiError,
// so anything else is a bug in the caller and reads as an internal error.
export function asApiError(err: unknown): ApiError {
  if (err instanceof ApiError) {
    return err;
  }
  return new ApiError(0, "internal", err instanceof Error ? err.message : String(err));
}

let unauthorizedHandler: (() => void) | undefined;

// setUnauthorizedHandler registers what happens when the session ends during
// use: a 401 with code unauthorized on any call (sign-in answers
// invalid_credentials instead, so it never triggers it).
export function setUnauthorizedHandler(handler: (() => void) | undefined): void {
  unauthorizedHandler = handler;
}

type Init = {
  method?: "GET" | "POST";
  body?: unknown;
};

// apiFetch calls the API and returns the parsed body, undefined for 204. The
// caller names the type from api-types.ts; the server is the authority on it.
export async function apiFetch<T>(path: string, init: Init = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  let body: string | undefined;
  if (init.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(init.body);
  }

  let res: Response;
  try {
    res = await fetch(basePath + path, {
      method: init.method ?? "GET",
      headers,
      body,
      credentials: "same-origin",
    });
  } catch {
    throw new ApiError(0, "network", "The server did not answer.");
  }

  if (res.ok) {
    if (res.status === 204) {
      return undefined as T;
    }
    try {
      return (await res.json()) as T;
    } catch {
      throw new ApiError(
        res.status,
        "internal",
        "The server's answer is not JSON.",
        [],
        requestIdOf(res),
      );
    }
  }

  const err = await readError(res);
  if (err.status === 401 && err.code === "unauthorized") {
    unauthorizedHandler?.();
  }
  throw err;
}

async function readError(res: Response): Promise<ApiError> {
  let body: unknown;
  try {
    body = await res.json();
  } catch {
    body = undefined;
  }
  if (isEnvelope(body)) {
    const e = body.error;
    return new ApiError(
      res.status,
      e.code,
      e.message,
      e.details ?? [],
      e.request_id || requestIdOf(res),
    );
  }
  return new ApiError(
    res.status,
    "internal",
    `The server answered ${res.status} without an error body.`,
    [],
    requestIdOf(res),
  );
}

function requestIdOf(res: Response): string | undefined {
  return res.headers.get("X-Request-Id") ?? undefined;
}

function isEnvelope(body: unknown): body is ErrorEnvelope {
  if (typeof body !== "object" || body === null || !("error" in body)) {
    return false;
  }
  const e: unknown = body.error;
  return (
    typeof e === "object" &&
    e !== null &&
    "code" in e &&
    typeof e.code === "string" &&
    "message" in e &&
    typeof e.message === "string" &&
    (!("details" in e) || e.details === undefined || Array.isArray(e.details))
  );
}
