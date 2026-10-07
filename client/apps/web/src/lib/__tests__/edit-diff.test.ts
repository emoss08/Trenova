import { describe, expect, it } from "vitest";
import { isPlainList, listDelta, previewValue } from "../edit-diff";

describe("listDelta", () => {
  it("names what was added and removed", () => {
    expect(listDelta(["a", "b", "c"], ["b", "c", "d"])).toEqual({ added: ["d"], removed: ["a"] });
  });
});

describe("isPlainList", () => {
  it("accepts lists of strings and numbers only", () => {
    expect(isPlainList(["a", 1])).toBe(true);
    expect(isPlainList([{ a: 1 }])).toBe(false);
    expect(isPlainList("a")).toBe(false);
  });
});

describe("previewValue", () => {
  it("writes each kind of value in a line", () => {
    expect(previewValue(true)).toBe("On");
    expect(previewValue(false)).toBe("Off");
    expect(previewValue(["a", "b"])).toBe("2 selected");
    expect(previewValue("")).toBe("Empty");
    expect(previewValue(null)).toBe("Empty");
    expect(previewValue({ a: 1 })).toBe("Changed");
    expect(previewValue("x".repeat(60))).toHaveLength(41);
  });
});
