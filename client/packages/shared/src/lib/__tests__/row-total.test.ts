import { describe, expect, it } from "vitest";
import { formatRowTotal, rowTotalOf } from "../row-total";

describe("formatRowTotal", () => {
  it("writes a counted total with every digit", () => {
    expect(formatRowTotal(9412, true)).toBe("9,412");
  });

  it("writes an estimated total compactly", () => {
    expect(formatRowTotal(1_250_000, false)).toBe("1.3M");
    expect(formatRowTotal(48_200, false)).toBe("48.2K");
  });
});

describe("rowTotalOf", () => {
  it("keeps whether the server counted the total", () => {
    expect(rowTotalOf(42, true)).toEqual({ count: 42, exact: true });
    expect(rowTotalOf(1_250_000, false)).toEqual({ count: 1_250_000, exact: false });
  });

  it("reads a missing total as none counted", () => {
    expect(rowTotalOf(null, undefined)).toEqual({ count: 0, exact: true });
  });
});
