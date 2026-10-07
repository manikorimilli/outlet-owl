// @vitest-environment node
// The config loads esbuild, which refuses jsdom's TextEncoder.
import { describe, expect, it } from "vitest";
import config from "../../vite.config";

// The string shorthand ("/api": "http://localhost:8080") turns changeOrigin
// on, which rewrites Host and makes the server refuse writes from browsers
// that send no Sec-Fetch-Site (web LLD sections 2 and 5).
describe("the Vite proxy", () => {
  it("forwards /api with changeOrigin false", () => {
    expect(config.server?.proxy?.["/api"]).toEqual({
      target: "http://localhost:8080",
      changeOrigin: false,
    });
  });
});
