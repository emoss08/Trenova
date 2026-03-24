import { describe, expect, it } from "vitest";
import { findDuplicateIds } from "@/lib/utils";

describe("findDuplicateIds", () => {
  it("returns empty set for empty array", () => {
    expect(findDuplicateIds([], (x: any) => x?.id)).toEqual(new Set());
  });

  it("returns empty set when no duplicates exist", () => {
    const items = [{ id: "a" }, { id: "b" }, { id: "c" }];
    expect(findDuplicateIds(items, (x) => x.id)).toEqual(new Set());
  });

  it("detects a single duplicate", () => {
    const items = [{ id: "a" }, { id: "b" }, { id: "a" }];
    expect(findDuplicateIds(items, (x) => x.id)).toEqual(new Set(["a"]));
  });

  it("detects multiple duplicates", () => {
    const items = [{ id: "a" }, { id: "b" }, { id: "a" }, { id: "b" }, { id: "c" }];
    expect(findDuplicateIds(items, (x) => x.id)).toEqual(new Set(["a", "b"]));
  });

  it("skips items with undefined or empty ids", () => {
    const items = [{ id: undefined }, { id: "" }, { id: "a" }, { id: "a" }];
    expect(findDuplicateIds(items, (x) => x.id)).toEqual(new Set(["a"]));
  });

  it("triple occurrence only appears once in the set", () => {
    const items = [{ id: "x" }, { id: "x" }, { id: "x" }];
    const result = findDuplicateIds(items, (x) => x.id);
    expect(result).toEqual(new Set(["x"]));
    expect(result.size).toBe(1);
  });
});
