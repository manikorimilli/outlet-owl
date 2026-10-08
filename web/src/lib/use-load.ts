import { useCallback, useEffect, useEffectEvent, useState } from "react";
import { asApiError, type ApiError } from "./api";

export type Load<T> =
  { status: "loading" } | { status: "error"; error: ApiError } | { status: "ready"; data: T };

// useLoad runs load whenever key changes or reload is called. An answer is
// kept only for the request it belongs to, so an older answer never shows
// under newer filters; "loading" is derived, not set.
export function useLoad<T>(key: string, load: () => Promise<T>): [Load<T>, () => void] {
  const [nonce, setNonce] = useState(0);
  const token = `${key}#${nonce}`;
  const [result, setResult] = useState<{ token: string; state: Load<T> } | null>(null);
  const start = useEffectEvent(load);

  useEffect(() => {
    let live = true;
    start().then(
      (data) => live && setResult({ token, state: { status: "ready", data } }),
      (err: unknown) =>
        live && setResult({ token, state: { status: "error", error: asApiError(err) } }),
    );
    return () => {
      live = false;
    };
  }, [token]);

  const reload = useCallback(() => setNonce((n) => n + 1), []);
  return [result?.token === token ? result.state : { status: "loading" }, reload];
}
