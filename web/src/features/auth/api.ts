import { apiFetch } from "../../lib/api";
import type { components } from "../../lib/api-types";

export type Me = components["schemas"]["Me"];
export type LoginRequest = components["schemas"]["LoginRequest"];

export function getMe(): Promise<Me> {
  return apiFetch<Me>("/me");
}

export function login(credentials: LoginRequest): Promise<Me> {
  return apiFetch<Me>("/auth/login", { method: "POST", body: credentials });
}

export function logout(): Promise<void> {
  return apiFetch<void>("/auth/logout", { method: "POST" });
}
