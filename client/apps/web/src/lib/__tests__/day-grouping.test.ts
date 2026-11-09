import { describe, expect, it } from "vitest";
import { dayBounds, dayGroupScope, dayKey, NO_DAY } from "../day-grouping";

const CHICAGO = "America/Chicago";

describe("dayKey and dayBounds", () => {
  it("read the day in the given zone, not UTC's", () => {
    expect(dayKey(Date.UTC(2026, 9, 8, 3, 0) / 1000, CHICAGO)).toBe("2026-10-07");
    expect(dayKey(Date.UTC(2026, 9, 8, 3, 0) / 1000, "UTC")).toBe("2026-10-08");
    expect(dayKey(null, CHICAGO)).toBe(NO_DAY);
  });

  it("cover a 25-hour day when the clocks fall back", () => {
    expect(dayBounds("2026-11-01", CHICAGO)).toEqual({
      start: Date.UTC(2026, 10, 1, 5) / 1000,
      end: Date.UTC(2026, 10, 2, 6) / 1000,
    });
    expect(dayBounds("not-a-day", CHICAGO)).toBeNull();
  });
});

describe("dayGroupScope", () => {
  it("keeps only the rows outside each collapsed day, and only dated rows when no-day collapses", () => {
    expect(dayGroupScope("timestamp", ["2026-11-01", NO_DAY], CHICAGO)).toEqual({
      fieldFilters: [{ field: "timestamp", operator: "isnotnull", value: null }],
      filterGroups: [
        {
          filters: [
            { field: "timestamp", operator: "lt", value: Date.UTC(2026, 10, 1, 5) / 1000 },
            { field: "timestamp", operator: "gte", value: Date.UTC(2026, 10, 2, 6) / 1000 },
            { field: "timestamp", operator: "isnull", value: null },
          ],
        },
      ],
    });
  });
});
