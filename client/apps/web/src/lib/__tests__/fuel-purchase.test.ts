import { describe, expect, it } from "vitest";
import { computeFuelTotal, purchaseQuarterLabel } from "../fuel-purchase";

describe("computeFuelTotal", () => {
  it("multiplies quantity by unit price to two places, half up", () => {
    expect(computeFuelTotal("100.5", "3.999")).toBe("401.90");
    expect(computeFuelTotal("120", "4.1295")).toBe("495.54");
    expect(computeFuelTotal("1", "0.005")).toBe("0.01");
  });

  it("returns a zero total for a zero quantity rather than nothing", () => {
    expect(computeFuelTotal("0", "3.50")).toBe("0.00");
  });

  it("returns null when either side is blank, missing or not a decimal", () => {
    expect(computeFuelTotal("", "3.50")).toBeNull();
    expect(computeFuelTotal("100", "")).toBeNull();
    expect(computeFuelTotal(null, "3.50")).toBeNull();
    expect(computeFuelTotal("100", undefined)).toBeNull();
    expect(computeFuelTotal("abc", "3.50")).toBeNull();
    expect(computeFuelTotal("100", "3.5.0")).toBeNull();
  });

  it("tolerates surrounding whitespace the way the inputs deliver it", () => {
    expect(computeFuelTotal(" 10 ", " 2.5 ")).toBe("25.00");
  });
});

describe("purchaseQuarterLabel", () => {
  it("names the IFTA quarter a purchase falls in, in the given timezone", () => {
    expect(purchaseQuarterLabel(Date.UTC(2026, 0, 15) / 1000, "UTC")).toBe("Q1 2026");
    expect(purchaseQuarterLabel(Date.UTC(2026, 5, 30, 23, 59) / 1000, "UTC")).toBe("Q2 2026");
    expect(purchaseQuarterLabel(Date.UTC(2026, 6, 1) / 1000, "UTC")).toBe("Q3 2026");
    expect(purchaseQuarterLabel(Date.UTC(2026, 11, 31) / 1000, "UTC")).toBe("Q4 2026");
  });

  it("places a purchase by the organization's local calendar, not UTC", () => {
    expect(purchaseQuarterLabel(Date.UTC(2026, 6, 1, 3, 0) / 1000, "America/Chicago")).toBe(
      "Q2 2026",
    );
  });
});
