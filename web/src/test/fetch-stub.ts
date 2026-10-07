// fetch replaced per test (tenet 5: no test reaches the network). Each route
// is "METHOD /api/v1/path"; a call no route names fails the test.
import { vi } from "vitest";
import type { Me } from "../features/auth/api";

export type Reply = { status: number; body?: unknown; headers?: Record<string, string> };
type Handler = Reply | "network" | ((init: RequestInit) => Reply | Promise<Reply>);

export function stubFetch(routes: Record<string, Handler>) {
  const fetchStub = vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const key = `${init.method ?? "GET"} ${String(input)}`;
    const handler = routes[key];
    if (handler === undefined) {
      throw new Error(`unexpected request ${key}`);
    }
    if (handler === "network") {
      throw new TypeError("Failed to fetch");
    }
    const reply = typeof handler === "function" ? await handler(init) : handler;
    const body = reply.body === undefined ? null : JSON.stringify(reply.body);
    return new Response(body, {
      status: reply.status,
      headers: { "Content-Type": "application/json", ...reply.headers },
    });
  });
  vi.stubGlobal("fetch", fetchStub);
  return fetchStub;
}

export function errorReply(
  status: number,
  code: string,
  message: string,
  details: { field: string; reason: string }[] = [],
): Reply {
  return { status, body: { error: { code, message, details, request_id: "req-test-1" } } };
}

export const unauthorized = errorReply(401, "unauthorized", "Sign in to continue.");

export const adminMe: Me = {
  id: 1,
  email: "ritika.rao@example.in",
  name: "Ritika Rao",
  role: "brand_admin",
  outlet: null,
  brand: { name: "Chai Point Demo", timezone: "Asia/Kolkata" },
};

export const managerMe: Me = {
  id: 2,
  email: "arjun.mehta@example.in",
  name: "Arjun Mehta",
  role: "outlet_manager",
  outlet: { id: 2, name: "Indiranagar" },
  brand: { name: "Chai Point Demo", timezone: "Asia/Kolkata" },
};
