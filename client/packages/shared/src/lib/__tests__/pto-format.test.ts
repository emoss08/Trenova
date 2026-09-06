import { describe, expect, it } from "vitest";
import { formatPtoDayTotal, formatPtoDays } from "../pto";

describe("formatPtoDays", () => {
  it("holds ledger precision", () => {
    expect(formatPtoDays("12.5")).toBe("12.50");
    expect(formatPtoDays("-0.25")).toBe("-0.25");
  });

  // Decimals cross the wire as strings. A value the server never sends should
  // still be shown rather than turned into "NaN" on the page.
  it("passes a non-numeric value through untouched", () => {
    expect(formatPtoDays("")).toBe("0.00");
    expect(formatPtoDays("unknown")).toBe("unknown");
  });
});

describe("formatPtoDayTotal", () => {
  it("keeps whole days whole and a fraction to one place", () => {
    expect(formatPtoDayTotal("312.00")).toBe("312");
    expect(formatPtoDayTotal("312.50")).toBe("312.5");
    expect(formatPtoDayTotal("312.46")).toBe("312.5");
  });

  it("groups a large total", () => {
    expect(formatPtoDayTotal("12480")).toBe("12,480");
  });

  it("passes a non-numeric value through untouched", () => {
    expect(formatPtoDayTotal("unknown")).toBe("unknown");
  });
});
