import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The Go server serves the built files from the same origin (ADR-0002); in
// development Vite forwards /api to it on :8080.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { "/api": "http://localhost:8080" },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
