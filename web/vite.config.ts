import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The Go server serves the built files from the same origin (ADR-0002); in
// development Vite forwards /api to it on :8080.
//
// changeOrigin stays false: the browser's Host (localhost:5173) must reach the
// server with its Origin, or the server's cross-origin check refuses writes
// from browsers that send no Sec-Fetch-Site (phase 1 server LLD, section 5).
// The string shorthand would turn changeOrigin on.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: false },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
