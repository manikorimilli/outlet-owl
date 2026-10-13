import { describe, expect, it } from "vitest";
import { demoAdmin, demoSignIn } from "./demo";

describe("demoSignIn", () => {
  it("prefills quietly under the dev server", () => {
    expect(demoSignIn({ MODE: "development" })).toEqual({ account: demoAdmin, note: false });
  });

  it("prefills and says it is a demo when the build asks for it", () => {
    expect(demoSignIn({ MODE: "production", VITE_DEMO_PREFILL: "true" })).toEqual({
      account: demoAdmin,
      note: true,
    });
  });

  it("leaves a normal build and the tests empty", () => {
    expect(demoSignIn({ MODE: "production" })).toEqual({ account: null, note: false });
    expect(demoSignIn({ MODE: "test" })).toEqual({ account: null, note: false });
    expect(demoSignIn({ MODE: "production", VITE_DEMO_PREFILL: "yes" }).account).toBeNull();
  });
});
