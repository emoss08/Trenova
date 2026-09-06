import { describe, expect, it } from "vitest";
import { ptoDays } from "../date";

const DAY = 86_400;
// Monday 2026-03-02 00:00 UTC
const MONDAY = Date.UTC(2026, 2, 2) / 1000;

describe("ptoDays", () => {
  it("counts every calendar day inclusively when weekends count", () => {
    expect(ptoDays(MONDAY, MONDAY + DAY * 6, true, "UTC")).toBe(7);
  });

  it("skips Saturday and Sunday when weekends do not count", () => {
    expect(ptoDays(MONDAY, MONDAY + DAY * 6, false, "UTC")).toBe(5);
  });

  it("treats a same-day request as one day and a reversed range as zero", () => {
    expect(ptoDays(MONDAY, MONDAY + 3600, false, "UTC")).toBe(1);
    expect(ptoDays(MONDAY + DAY, MONDAY, true, "UTC")).toBe(0);
    expect(ptoDays(0, MONDAY, true, "UTC")).toBe(0);
  });
});
