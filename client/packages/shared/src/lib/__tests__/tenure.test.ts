import { describe, expect, it } from "vitest";
import { formatTenure, tenureParts } from "../tenure";

const DAY = 86_400;
const now = Date.UTC(2026, 8, 2) / 1000;

describe("tenureParts", () => {
  it("splits service into whole years, months and days as of now", () => {
    expect(tenureParts(Date.UTC(2024, 5, 15) / 1000, null, now)).toEqual({
      years: 2,
      months: 2,
      days: 18,
      totalDays: 809,
    });
  });

  it("stops at the termination date and ignores one in the future", () => {
    const hire = Date.UTC(2025, 0, 1) / 1000;
    expect(tenureParts(hire, Date.UTC(2025, 6, 1) / 1000, now).months).toBe(6);
    expect(tenureParts(hire, now + 30 * DAY, now)).toEqual(tenureParts(hire, null, now));
  });

  it("is zero without a hire date or when hired in the future", () => {
    expect(tenureParts(0, null, now).totalDays).toBe(0);
    expect(tenureParts(now + DAY, null, now).totalDays).toBe(0);
  });
});

describe("formatTenure", () => {
  it("prefers the two most significant units", () => {
    expect(formatTenure(Date.UTC(2024, 5, 15) / 1000, null, now)).toBe("2y 2m");
    expect(formatTenure(Date.UTC(2026, 3, 2) / 1000, null, now)).toBe("5m");
    expect(formatTenure(Date.UTC(2026, 7, 20) / 1000, null, now)).toBe("13d");
    expect(formatTenure(Date.UTC(2026, 8, 2) / 1000, null, now)).toBe("Today");
    expect(formatTenure(Date.UTC(2020, 8, 2) / 1000, null, now)).toBe("6y");
  });

  it("returns a dash when there is nothing to measure", () => {
    expect(formatTenure(null, null, now)).toBe("—");
    expect(formatTenure(0, null, now)).toBe("—");
  });
});
