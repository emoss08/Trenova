import { describe, expect, it } from "vitest";
import {
  basicStandings,
  eventTotals,
  isWindowMonths,
  ratingSegments,
  terminalStandings,
  trendPeak,
  trendRows,
} from "../fleet-safety-console";

describe("ratingSegments", () => {
  // A bar that drops "At risk" when it empties reads as a category that does
  // not exist; every rating stays on the legend, in scale order.
  it("returns all four ratings in scale order, zeros included", () => {
    expect(
      ratingSegments([
        { rating: "AtRisk", workers: 2 },
        { rating: "Excellent", workers: 15 },
      ]),
    ).toEqual([
      { rating: "Excellent", label: "Excellent", workers: 15 },
      { rating: "Good", label: "Good", workers: 0 },
      { rating: "Watch", label: "Watch", workers: 0 },
      { rating: "AtRisk", label: "At risk", workers: 2 },
    ]);
  });
});

describe("basicStandings", () => {
  const basic = (basic: string, weightedScore: number) => ({
    basic,
    violations: 0,
    events: 0,
    weightedScore,
    outOfService: 0,
    inferred: false,
  });

  it("reads each BASIC against the fleet's own worst, never a percentile", () => {
    const standings = basicStandings([
      basic("UnsafeDriving", 60),
      basic("HOSCompliance", 30),
      basic("DriverFitness", 6),
      basic("CrashIndicator", 0),
    ]);
    expect(standings.map((row) => [row.basic.basic, row.tone, row.share])).toEqual([
      ["UnsafeDriving", "critical", 100],
      ["HOSCompliance", "warning", 50],
      ["DriverFitness", "muted", 10],
      ["CrashIndicator", "muted", 0],
    ]);
  });

  it("floors a tiny score at a visible sliver and a zero fleet at nothing", () => {
    expect(basicStandings([basic("A", 100), basic("B", 1)])[1].share).toBe(3);
    expect(basicStandings([basic("A", 0)])[0].share).toBe(0);
  });
});

describe("eventTotals", () => {
  it("adds every kind together", () => {
    expect(
      eventTotals([
        { kind: "Accident", events: 3, points: 18, preventable: 2, outOfService: 0, open: 1 },
        { kind: "Inspection", events: 5, points: 4, preventable: 0, outOfService: 1, open: 0 },
      ]),
    ).toEqual({ events: 8, preventable: 2, open: 1, outOfService: 1, points: 22 });
  });
});

describe("trend", () => {
  const point = (periodStart: number, events: number) => ({
    periodStart,
    events,
    accidents: 0,
    preventable: 0,
    citations: 0,
    inspections: 0,
    outOfService: 0,
    points: 0,
  });

  it("orders the months oldest first and labels them in UTC", () => {
    const rows = trendRows([
      point(Date.UTC(2026, 7, 1) / 1000, 5),
      point(Date.UTC(2026, 5, 1) / 1000, 2),
    ]);
    expect(rows.map((row) => [row.label, row.events])).toEqual([
      ["Jun", 2],
      ["Aug", 5],
    ]);
  });

  it("finds the busiest month and nothing for an empty window", () => {
    expect(trendPeak([point(1, 2), point(2, 7), point(3, 7)])?.periodStart).toBe(2);
    expect(trendPeak([])).toBeNull();
  });
});

describe("terminalStandings", () => {
  const terminal = (code: string, workers: number, atRisk: number, watch: number) => ({
    fleetCodeId: `fc_${code}`,
    code,
    workers,
    atRisk,
    watch,
    averageScore: 80,
  });

  // The question after "how is the fleet" is "is it one yard"; the yard with
  // the most drivers at risk comes first, whatever its size.
  it("puts the terminal with the most at-risk drivers first, with its share of the fleet", () => {
    const standings = terminalStandings(
      [terminal("BIG", 40, 0, 1), terminal("SMALL", 5, 2, 0), terminal("MID", 15, 1, 3)],
      60,
    );
    expect(standings.map((row) => [row.terminal.code, row.flagged, row.share])).toEqual([
      ["SMALL", 2, 8],
      ["MID", 4, 25],
      ["BIG", 1, 67],
    ]);
  });
});

describe("isWindowMonths", () => {
  it("accepts only the windows the control offers", () => {
    expect(isWindowMonths("12")).toBe(true);
    expect(isWindowMonths("9")).toBe(false);
  });
});
