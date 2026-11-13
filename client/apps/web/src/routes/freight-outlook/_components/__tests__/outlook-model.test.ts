import {
  area,
  cleveland,
  kansasCity,
  move,
  outlook,
  slotStarts,
  windows,
} from "@/lib/graphql/__tests__/freight-outlook-fixtures";
import type { FreightPace } from "@/lib/graphql/freight-outlook";
import { describe, expect, it } from "vitest";
import {
  balanceFigure,
  balanceOf,
  daysInRange,
  horizonSlots,
  laneGroups,
  laneOf,
  moveKey,
  moveTrail,
  movesIn,
  paceReadings,
  paceStanding,
  peakOf,
  reachBounds,
  positionAmong,
  windowIndexAt,
  totalsOf,
  viewIndexes,
} from "../outlook-model";

describe("outlook model", () => {
  it("reads twelve, twenty-four or forty-eight hours as three, six or twelve windows", () => {
    expect(horizonSlots("12")).toBe(3);
    expect(horizonSlots("24")).toBe(6);
    expect(horizonSlots("48")).toBe(12);
    expect(viewIndexes(3, null)).toEqual([0, 1, 2]);
    expect(viewIndexes(12, 7)).toEqual([7]);
  });

  it("signs a window's balance: minus for trucks short, plus for trucks to spare", () => {
    expect(balanceOf({ short: 2.8, spare: 0 })).toEqual({ kind: "short", level: 3, amount: 2.8 });
    expect(balanceFigure(balanceOf({ short: 2.8, spare: 0 }))).toBe("−3");
    expect(balanceOf({ short: 0.2, spare: 1.6 })).toEqual({ kind: "spare", level: 2, amount: 1.6 });
    expect(balanceFigure(balanceOf({ short: 0.2, spare: 1.6 }))).toBe("+2");
    expect(balanceOf({ short: 0.4, spare: 0.4 })).toEqual({ kind: "even", level: 0, amount: 0 });
    expect(balanceFigure(balanceOf({ short: 0.4, spare: 0.4 }))).toBe("");
  });

  it("puts an area in the lane of its worst window in the reading", () => {
    expect(peakOf(cleveland, [0, 1, 2])).toEqual({ kind: "short", amount: 2.8, index: 1 });
    expect(laneOf(peakOf(cleveland, [0, 1, 2]))).toBe("short");
    expect(laneOf(peakOf(cleveland, [0]))).toBe("even");
    const tight = area(1, "Memphis, TN", { 3: { short: 0.9 }, 4: { spare: 2 } });
    expect(laneOf(peakOf(tight, [3, 4]))).toBe("tight");
    expect(peakOf(kansasCity, [0, 1])).toEqual({ kind: "spare", amount: 3.3, index: 0 });
    expect(laneOf(peakOf(kansasCity, [0, 1]))).toBe("spare");
  });

  it("totals loads with no truck over every window and idle drivers at each area's worst", () => {
    const totals = totalsOf(
      outlook({
        network: outlook().network.map((slot, index) =>
          index === 2 ? { ...slot, short: 4.2 } : slot,
        ),
      }),
      [0, 1, 2],
    );

    expect(totals).toEqual({
      uncovered: 3,
      idle: 3,
      shortAreas: 1,
      spareAreas: 1,
      worthCents: 84_520,
      moves: 1,
      networkShort: 4.2,
      networkShortIndex: 2,
    });
  });

  it("keeps the moves for a reading, and a pinned window's neighbours", () => {
    const moves = [
      move({ slotStart: slotStarts[1] }),
      move({ slotStart: slotStarts[4], workerName: "Ben Ito" }),
      move({ slotStart: slotStarts[9], workerName: "Late" }),
    ];

    expect(movesIn(moves, slotStarts, [0, 1, 2]).map((m) => m.workerName)).toEqual(["Ana Ruiz"]);
    expect(movesIn(moves, slotStarts, [3]).map((m) => m.workerName)).toEqual(["Ben Ito"]);
    expect(movesIn(moves, slotStarts, [])).toEqual([]);
  });

  it("measures a customer against its own pace, the ones behind first", () => {
    const pace = (name: string, usual: number, byNow: number, tendered: number): FreightPace => ({
      customerId: `cus_${name}`,
      customerName: name,
      cell: 1,
      label: "Cleveland, OH",
      slots: slotStarts.map((start, index) => ({
        start,
        usual: index === 1 ? usual : 0,
        byNow: index === 1 ? byNow : 0,
        tendered: index === 1 ? tendered : 0,
      })),
    });

    const readings = paceReadings(
      [pace("Steady", 3, 2, 2), pace("Busy", 4, 1, 4), pace("Slow", 5, 3, 1)],
      [0, 1, 2],
    );

    expect(readings.map((r) => [r.pace.customerName, r.standing])).toEqual([
      ["Slow", "Behind"],
      ["Busy", "Ahead"],
      ["Steady", "OnPace"],
    ]);
    expect(readings[0].busiestStart).toBe(slotStarts[1]);
    expect(paceStanding(0.8, 2, 0)).toBe("OnPace");
  });

  it("finds the window that holds a moment, and nothing outside the outlook", () => {
    expect(windowIndexAt(windows, slotStarts[5])).toBe(5);
    expect(windowIndexAt(windows, slotStarts[5] + 3 * 3_600)).toBe(5);
    expect(windowIndexAt(windows, slotStarts[6] - 1)).toBe(5);
    expect(windowIndexAt(windows, slotStarts[0] - 1)).toBeNull();
    expect(windowIndexAt(windows, slotStarts[11] + 4 * 3_600)).toBeNull();
    expect(windowIndexAt([], slotStarts[0])).toBeNull();
  });

  it("reads a window that is not four hours long by its own end", () => {
    const hour = 3_600;
    // The night the clocks go back: the first window runs five hours, the rest four.
    const longNight = [
      { start: 0, end: 5 * hour },
      { start: 5 * hour, end: 9 * hour },
      { start: 9 * hour, end: 13 * hour },
    ];

    expect(windowIndexAt(longNight, 4.5 * hour)).toBe(0);
    expect(windowIndexAt(longNight, 5 * hour)).toBe(1);
    expect(windowIndexAt(longNight, 8.5 * hour)).toBe(1);
    expect(positionAmong(longNight, 2.5 * hour)).toBeCloseTo(0.5);
    expect(positionAmong(longNight, 7 * hour)).toBeCloseTo(1.5);
    expect(positionAmong(longNight, -hour)).toBe(0);
    expect(positionAmong(longNight, 20 * hour)).toBe(3);
    expect(positionAmong([], hour)).toBe(0);
  });
});

describe("moves on the board", () => {
  it("draws a send from the area and window its driver frees up in", () => {
    const trail = moveTrail(move({}), windows);

    expect(trail).toEqual({
      toCell: cleveland.cell,
      toIndex: 1,
      fromCell: kansasCity.cell,
      fromIndex: 0,
    });
  });

  it("starts a keep from its partner, and a driver free before the outlook at its first window", () => {
    const keep = move({
      kind: "Keep",
      fromCell: cleveland.cell,
      partnerCell: kansasCity.cell,
      partnerFreeAt: slotStarts[0] - 3_600,
    });

    expect(moveTrail(keep, windows)).toEqual({
      toCell: cleveland.cell,
      toIndex: 1,
      fromCell: kansasCity.cell,
      fromIndex: 0,
    });
  });

  it("gives a call, and a move that stays in its own area, nowhere to start from", () => {
    const call = move({ kind: "Call", fromCell: 0, customerId: "cus_northstar" });
    const local = move({ fromCell: cleveland.cell });

    expect(moveTrail(call, windows).fromCell).toBeNull();
    expect(moveTrail(local, windows).fromCell).toBeNull();
  });

  it("names a move by its kind, driver or customer, target and window", () => {
    expect(moveKey(move({}))).toBe(`Send:wrk_ana:${cleveland.cell}:${slotStarts[1]}`);
    expect(moveKey(move({ kind: "Call", workerId: null, customerId: "cus_northstar" }))).toBe(
      `Call:cus_northstar:${cleveland.cell}:${slotStarts[1]}`,
    );
  });
});

describe("board groups and the record", () => {
  it("groups areas by what they need, short first, worst first, and drops empty groups", () => {
    const worse = area(999, "Edison, NJ", { 1: { short: 3.4 } });
    const groups = laneGroups([cleveland, kansasCity, worse], [0, 1, 2]);

    expect(groups.map((group) => group.lane)).toEqual(["short", "spare"]);
    expect(groups[0].rows.map((row) => row.area.label)).toEqual(["Edison, NJ", "Cleveland, OH"]);
  });

  it("counts a day inside its range only when the loads landed between its ends", () => {
    expect(
      daysInRange([
        { low: 10, high: 20, actual: 10 },
        { low: 10, high: 20, actual: 20 },
        { low: 10, high: 20, actual: 21 },
        { low: 10, high: 20, actual: 9 },
      ]),
    ).toBe(2);
  });

  it("frames the area's tile alone when no driver is measured against it", () => {
    const tile = { south: 32.25, west: -97.03, north: 33.14, east: -95.63 };
    expect(reachBounds(tile, [])).toEqual(tile);
  });

  it("widens the view to every driver outside the tile and keeps the tile's own edges otherwise", () => {
    const tile = { south: 32.25, west: -97.03, north: 33.14, east: -95.63 };
    expect(
      reachBounds(tile, [
        { latitude: 29.76, longitude: -95.37 },
        { latitude: 35.47, longitude: -97.52 },
        { latitude: 32.7, longitude: -96.5 },
      ]),
    ).toEqual({ south: 29.76, west: -97.52, north: 35.47, east: -95.37 });
  });
});
