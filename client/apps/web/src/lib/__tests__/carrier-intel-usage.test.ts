import { describe, expect, it } from "vitest";
import {
  dailyUsageSeries,
  recentUsageMonths,
  shiftUsageMonth,
  spendCapProgress,
  summarizeUsageByDay,
  utcDayKeyToUnix,
} from "../carrier-intel-usage";
import type { CarrierIntelUsageSummary } from "../graphql/carrier-intel-settings";

/** Rows as the usage query returns them, endpoint and all. */
function daily(rows: CarrierIntelUsageSummary["daily"]) {
  return rows;
}

describe("carrier intelligence usage", () => {
  it("lists UTC month starts newest first, crossing the year boundary", () => {
    expect(recentUsageMonths(Date.UTC(2026, 1, 14, 23, 30) / 1000, 3)).toEqual([
      Date.UTC(2026, 1, 1) / 1000,
      Date.UTC(2026, 0, 1) / 1000,
      Date.UTC(2025, 11, 1) / 1000,
    ]);
  });

  it("reads a UTC day key as the start of that day", () => {
    expect(utcDayKeyToUnix(20260916)).toBe(Date.UTC(2026, 8, 16) / 1000);
  });

  it("totals every endpoint for a day, newest day first", () => {
    expect(
      summarizeUsageByDay(
        daily([
          { day: 20260915, endpoint: "lookup", calls: 3, billableUnits: 3, estimatedCost: "1.50" },
          { day: 20260916, endpoint: "lookup", calls: 2, billableUnits: 2, estimatedCost: "1.00" },
          { day: 20260916, endpoint: "monitor", calls: 1, billableUnits: 4, estimatedCost: "0.25" },
        ]),
      ),
    ).toEqual([
      { day: 20260916, calls: 3, billableUnits: 6, estimatedCost: "1.25" },
      { day: 20260915, calls: 3, billableUnits: 3, estimatedCost: "1.50" },
    ]);
  });

  it("places spend against the soft and hard caps", () => {
    expect(spendCapProgress("10.00", null, 80).state).toBe("uncapped");
    expect(spendCapProgress("50.00", "100.00", 80)).toEqual({
      state: "within",
      percent: 50,
      softCapAmount: "80.00",
    });
    expect(spendCapProgress("85.00", "100.00", 80).state).toBe("soft");
    expect(spendCapProgress("120.00", "100.00", 80)).toMatchObject({
      state: "exceeded",
      percent: 100,
    });
  });

  it("moves a month start across year boundaries", () => {
    expect(shiftUsageMonth(Date.UTC(2026, 0, 1) / 1000, -1)).toBe(Date.UTC(2025, 11, 1) / 1000);
    expect(shiftUsageMonth(Date.UTC(2025, 11, 1) / 1000, 1)).toBe(Date.UTC(2026, 0, 1) / 1000);
  });

  it("fills every day of the month up to today, with zero cost on quiet days", () => {
    const monthStart = Date.UTC(2026, 8, 1) / 1000;
    const now = Date.UTC(2026, 8, 3, 15) / 1000;
    const series = dailyUsageSeries(
      daily([
        { day: 20260901, endpoint: "lookup", calls: 2, billableUnits: 2, estimatedCost: "1.00" },
        { day: 20260903, endpoint: "lookup", calls: 1, billableUnits: 1, estimatedCost: "0.50" },
        { day: 20260903, endpoint: "monitor", calls: 1, billableUnits: 3, estimatedCost: "0.25" },
      ]),
      monthStart,
      now,
    );

    expect(series.map((point) => [point.day, point.cost, point.calls])).toEqual([
      [20260901, 1, 2],
      [20260902, 0, 0],
      [20260903, 0.75, 2],
    ]);
  });

  it("covers the whole month for a month that has ended", () => {
    const series = dailyUsageSeries([], Date.UTC(2026, 1, 1) / 1000, Date.UTC(2026, 8, 1) / 1000);
    expect(series).toHaveLength(28);
  });
});
