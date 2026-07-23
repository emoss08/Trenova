import { addDays, startOfDay } from "date-fns";
import { describe, expect, it } from "vitest";
import {
  buildDayColumns,
  buildHourTicks,
  canvasWidthPx,
  getBarGeometry,
  getTimelineRange,
  packLanes,
  parseAnchorDate,
  secondsToX,
  serializeAnchorDate,
  shiftAnchor,
  ZOOM_CONFIGS,
} from "./time-scale";

const anchor = new Date(2026, 6, 20); // Mon Jul 20 2026, local time

describe("anchor parsing", () => {
  it("round-trips a serialized anchor", () => {
    const parsed = parseAnchorDate(serializeAnchorDate(anchor));
    expect(parsed.getTime()).toBe(startOfDay(anchor).getTime());
  });

  it("falls back to today for null and garbage input", () => {
    const today = startOfDay(new Date()).getTime();
    expect(parseAnchorDate(null).getTime()).toBe(today);
    expect(parseAnchorDate("not-a-date").getTime()).toBe(today);
  });
});

describe("getTimelineRange", () => {
  it("spans exactly the configured number of days", () => {
    const range = getTimelineRange(anchor, "3day");
    expect(range.start).toBe(Math.floor(startOfDay(anchor).getTime() / 1000));
    expect(range.end).toBe(Math.floor(addDays(startOfDay(anchor), 3).getTime() / 1000));
  });
});

describe("shiftAnchor", () => {
  it("moves by the zoom's day count in either direction", () => {
    expect(shiftAnchor(anchor, "week", 1).getTime()).toBe(addDays(anchor, 7).getTime());
    expect(shiftAnchor(anchor, "day", -1).getTime()).toBe(addDays(anchor, -1).getTime());
  });
});

describe("secondsToX / canvasWidthPx", () => {
  it("maps range boundaries to 0 and full width", () => {
    const range = getTimelineRange(anchor, "day");
    const width = canvasWidthPx(range, "day");
    expect(secondsToX(range.start, range, "day")).toBe(0);
    expect(secondsToX(range.end, range, "day")).toBe(width);
  });

  it("clamps out-of-range values", () => {
    const range = getTimelineRange(anchor, "day");
    expect(secondsToX(range.start - 5000, range, "day")).toBe(0);
    expect(secondsToX(range.end + 5000, range, "day")).toBe(canvasWidthPx(range, "day"));
  });

  it("positions noon halfway across a standard day", () => {
    const range = getTimelineRange(anchor, "day");
    const noon = range.start + 12 * 3600;
    expect(secondsToX(noon, range, "day")).toBeCloseTo(canvasWidthPx(range, "day") / 2, 5);
  });
});

describe("buildDayColumns", () => {
  it("produces one column per day covering the full range", () => {
    const range = getTimelineRange(anchor, "week");
    const columns = buildDayColumns(range, "week");
    expect(columns).toHaveLength(7);
    expect(columns[0].x).toBe(0);
    const last = columns[columns.length - 1];
    expect(last.x + last.width).toBeCloseTo(canvasWidthPx(range, "week"), 5);
  });

  it("flags weekends", () => {
    const range = getTimelineRange(anchor, "week"); // Mon..Sun
    const columns = buildDayColumns(range, "week");
    expect(columns.map((c) => c.isWeekend)).toEqual([
      false,
      false,
      false,
      false,
      false,
      true,
      true,
    ]);
  });
});

describe("buildHourTicks", () => {
  it("steps by the zoom's hour step and skips day boundaries", () => {
    const range = getTimelineRange(anchor, "day");
    const ticks = buildHourTicks(range, "day");
    expect(ticks).toHaveLength(11); // 02:00..22:00 every 2h
    expect(ticks[0].label).toBe("02:00");
    expect(ticks[ticks.length - 1].label).toBe("22:00");
  });

  it("returns nothing for week zoom", () => {
    const range = getTimelineRange(anchor, "week");
    expect(buildHourTicks(range, "week")).toHaveLength(0);
  });
});

describe("getBarGeometry", () => {
  it("clips spans that extend past the range and marks the clipped edges", () => {
    const range = getTimelineRange(anchor, "day");
    const geometry = getBarGeometry(range.start - 3600, range.start + 7200, range, "day");
    expect(geometry.left).toBe(0);
    expect(geometry.clippedStart).toBe(true);
    expect(geometry.clippedEnd).toBe(false);
    expect(geometry.width).toBeCloseTo(2 * ZOOM_CONFIGS.day.pxPerHour, 5);
  });

  it("enforces a minimum visual width for instant stops", () => {
    const range = getTimelineRange(anchor, "day");
    const t = range.start + 3600;
    const geometry = getBarGeometry(t, t, range, "day");
    expect(geometry.width).toBeGreaterThanOrEqual(14);
  });
});

describe("packLanes", () => {
  it("keeps non-overlapping spans in one lane", () => {
    const packing = packLanes([
      { start: 0, end: 10 },
      { start: 10, end: 20 },
      { start: 25, end: 30 },
    ]);
    expect(packing.laneCount).toBe(1);
    expect(packing.laneByIndex).toEqual([0, 0, 0]);
  });

  it("stacks overlapping spans into separate lanes and reuses freed lanes", () => {
    const packing = packLanes([
      { start: 0, end: 10 },
      { start: 5, end: 15 },
      { start: 6, end: 8 },
      { start: 12, end: 20 },
    ]);
    expect(packing.laneByIndex).toEqual([0, 1, 2, 0]);
    expect(packing.laneCount).toBe(3);
  });

  it("returns a single lane for empty input", () => {
    expect(packLanes([]).laneCount).toBe(1);
  });
});
