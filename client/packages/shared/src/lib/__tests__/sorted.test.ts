import { describe, expect, it } from "vitest";
import { lastAtOrBefore, sortedRange } from "../sorted";

const segments = [{ at: 2 }, { at: 5 }, { at: 5 }, { at: 9 }, { at: 14 }];
const at = (item: { at: number }) => item.at;

describe("sortedRange", () => {
  it("takes every item whose key falls between the bounds, both included", () => {
    expect(sortedRange(segments, 5, 9, at)).toEqual([{ at: 5 }, { at: 5 }, { at: 9 }]);
    expect(sortedRange(segments, 3, 4, at)).toEqual([]);
    expect(sortedRange(segments, 0, 100, at)).toEqual(segments);
    expect(sortedRange(segments, 14, Number.POSITIVE_INFINITY, at)).toEqual([{ at: 14 }]);
  });

  it("takes nothing from an empty list or a range that runs backwards", () => {
    expect(sortedRange([], 0, 10, at)).toEqual([]);
    expect(sortedRange(segments, 9, 5, at)).toEqual([]);
  });
});

describe("lastAtOrBefore", () => {
  it("finds the last item whose key is at or before the value", () => {
    expect(lastAtOrBefore(segments, 5, at)).toEqual({ at: 5 });
    expect(lastAtOrBefore(segments, 8, at)).toEqual({ at: 5 });
    expect(lastAtOrBefore(segments, 100, at)).toEqual({ at: 14 });
  });

  it("finds nothing before the first item", () => {
    expect(lastAtOrBefore(segments, 1, at)).toBeUndefined();
    expect(lastAtOrBefore([], 1, at)).toBeUndefined();
  });
});
