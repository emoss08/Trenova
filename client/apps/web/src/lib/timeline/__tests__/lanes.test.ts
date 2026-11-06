import { describe, expect, it } from "vitest";
import { laneGaps, nextStartsInRow, packLane } from "../lanes";
import { dayBoundaries } from "../time-scale";

const TUE = Date.UTC(2026, 10, 3) / 1000;
const HOUR = 3_600;
const at = (hours: number) => TUE + hours * HOUR;
const span = (id: string, from: number, to: number) => ({ id, start: at(from), end: at(to) });

describe("packLane", () => {
  it("stacks overlapping bars into rows and marks each one as double-booked", () => {
    const packed = packLane([span("a", 8, 14), span("b", 12, 18), span("c", 15, 20)]);

    expect(packed.rows).toBe(2);
    expect(packed.placed.map((p) => [p.item.id, p.row])).toEqual([
      ["a", 0],
      ["b", 1],
      ["c", 0],
    ]);
    expect([...packed.conflicts].sort()).toEqual(["a", "b", "c"]);
  });

  it("packs a lane that is not a driver's without calling its overlaps double-booked", () => {
    const packed = packLane([span("a", 8, 14), span("b", 12, 18)], { flagOverlaps: false });

    expect(packed.rows).toBe(2);
    expect(packed.conflicts.size).toBe(0);
  });

  it("keeps back-to-back bars on one row with no conflict", () => {
    const packed = packLane([span("a", 8, 12), span("b", 12, 16)]);

    expect(packed.rows).toBe(1);
    expect(packed.conflicts.size).toBe(0);
  });

  it("gives an empty lane one row, so it is still somewhere to drop work", () => {
    expect(packLane([]).rows).toBe(1);
  });
});

describe("nextStartsInRow", () => {
  it("bounds each bar by the next one in its own row, never by a bar in another row", () => {
    const packed = packLane([span("a", 8, 10), span("b", 9, 12), span("c", 11, 13)]);

    // a and c share row 0; b sits alone in row 1.
    expect(Object.fromEntries(nextStartsInRow(packed))).toEqual({ a: at(11) });
  });
});

describe("laneGaps", () => {
  it("labels free time between bars and the open time after the last one", () => {
    const gaps = laneGaps([span("a", 8, 11), span("b", 15, 17)], at(10), at(24));

    expect(gaps).toEqual([
      { start: at(11), end: at(15), open: false },
      { start: at(17), end: at(24), open: true },
    ]);
  });

  it("leaves out free time too short to read", () => {
    expect(laneGaps([span("a", 8, 11), span("b", 12, 23)], at(8), at(24))).toEqual([]);
  });

  it("frees a lane with nothing on it for the whole range", () => {
    expect(laneGaps([], at(10), at(24))).toEqual([{ start: at(10), end: at(24), open: true }]);
  });
});

describe("dayBoundaries", () => {
  it("starts each day at its own local midnight across a daylight-saving change", () => {
    // America/Chicago falls back on Sunday 1 November 2026: that day is 25 hours long.
    const satStart = Date.UTC(2026, 9, 31, 5) / 1000;
    const boundaries = dayBoundaries(satStart, 3, "America/Chicago");

    expect(boundaries).toEqual([
      satStart,
      satStart + 24 * HOUR,
      satStart + 49 * HOUR,
      satStart + 73 * HOUR,
    ]);
  });
});
