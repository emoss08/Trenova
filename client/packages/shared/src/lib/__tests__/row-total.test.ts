import { describe, expect, it } from "vitest";
import { formatRowTotal } from "../row-total";

describe("formatRowTotal", () => {
  it("writes a counted total with every digit", () => {
    expect(formatRowTotal(9412, true)).toBe("9,412");
  });

  it("writes an estimated total compactly", () => {
    expect(formatRowTotal(1_250_000, false)).toBe("1.3M");
    expect(formatRowTotal(48_200, false)).toBe("48.2K");
  });
});
