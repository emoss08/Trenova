import { describe, expect, it } from "vitest";
import { formatDayList, parseDayList } from "../day-list";

describe("parseDayList", () => {
  it("reads days separated by commas or spaces, earliest first, each once", () => {
    expect(parseDayList("30, 1,15 15")).toEqual([1, 15, 30]);
  });

  it("is empty for nothing typed", () => {
    expect(parseDayList("  ")).toEqual([]);
  });

  it("refuses anything that is not a whole number of days", () => {
    expect(parseDayList("1, 2.5")).toBeNull();
    expect(parseDayList("1, -3")).toBeNull();
    expect(parseDayList("one")).toBeNull();
  });
});

describe("formatDayList", () => {
  it("writes the days back as typed", () => {
    expect(formatDayList([1, 15, 30])).toBe("1, 15, 30");
    expect(formatDayList(null)).toBe("");
  });
});
