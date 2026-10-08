import { describe, expect, it } from "vitest";
import { formatInstant } from "./format";

describe("formatInstant", () => {
  it("dates an instant in the brand timezone, not UTC", () => {
    // 01:30 IST on 5 Oct is still 4 Oct in UTC.
    expect(formatInstant("2026-10-04T20:00:00Z", "Asia/Kolkata")).toBe("5 Oct 2026");
    expect(formatInstant("2026-10-04T20:00:00Z", "UTC")).toBe("4 Oct 2026");
    expect(formatInstant("2026-09-28T05:00:00Z", "Asia/Kolkata")).toBe("28 Sep 2026");
  });
});
