import { describe, expect, it } from "vitest";
import { formatStudioValue } from "../format";

describe("formatStudioValue", () => {
  it("prints numbers grouped, flags as words and text as typed", () => {
    expect(formatStudioValue(12345.678)).toBe("12,345.678");
    expect(formatStudioValue("742")).toBe("742");
    expect(formatStudioValue(true)).toBe("true");
    expect(formatStudioValue("EXP")).toBe("EXP");
    expect(formatStudioValue(null)).toBe("—");
    expect(formatStudioValue("")).toBe("—");
  });

  it("describes a list or a record by its size instead of dumping it", () => {
    expect(formatStudioValue([{ hazmat: false, weight: 0 }])).toBe("1 item");
    expect(formatStudioValue([1, 2, 3])).toBe("3 items");
    expect(formatStudioValue([])).toBe("empty list");
    expect(formatStudioValue({ city: "Austin", state: "TX" })).toBe("2 fields");
    expect(formatStudioValue({})).toBe("empty record");
  });
});
