import { describe, expect, it } from "vitest";
import { parseHourOverride } from "./use-local-hour";

describe("parseHourOverride", () => {
  it.each([
    ["0", 0],
    ["6", 6],
    ["17", 17],
    ["19.5", 19.5],
    ["23.99", 23.99],
    ["7:30", 7.5],
    ["16:45", 16.75],
    ["00:00", 0],
  ])("reads %j as hour %d", (value, hour) => {
    expect(parseHourOverride(value)).toBe(hour);
  });

  it.each([null, "", "24", "24:00", "-1", "7:60", "7:5", "noon", "17h", " 17", "1e1", "123"])(
    "is null for %j, so the clock is used",
    (value) => {
      expect(parseHourOverride(value)).toBeNull();
    },
  );
});
