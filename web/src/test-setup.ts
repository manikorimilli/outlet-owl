import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Vitest runs without globals, so Testing Library cannot clean up by itself;
// every test also gets its own fetch stub.
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
